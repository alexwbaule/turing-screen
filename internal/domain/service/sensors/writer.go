package sensors

import (
	"fmt"
	"time"

	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/utils"
	"golang.org/x/image/font"
)

// Default sizes per sensor type (max chars the formatted value can produce)
const (
	SizePercent = 4 // "100%"
	SizeHertz   = 7 // "5.20GHz"
	SizeTemp    = 4 // "100°C" (°C = 2 runes but font renders as ~2 chars)
	SizeBytes   = 7 // "32.0GiB"
	SizeSpeed   = 8 // "999 GB/s"
	SizeDate    = 8 // "15:04:05"
	SizePower   = 4 // "999W"
	SizeDefault = 8 // fallback
)

func BuildRadial(builder *renderer.Builder, mesurement float64, radial *theme.Radial) (*device.ImageProcess, int, int) {
	img := builder.DrawRadialProgressBar(mesurement, radial)
	return device.NewImageProcess(img), radial.X - radial.Radius, radial.Y - radial.Radius
}

func BuildGraph(builder *renderer.Builder, mesurement float64, graph *theme.Graph) (*device.ImageProcess, int, int) {
	img := builder.DrawProgressBar(mesurement, graph)
	return device.NewImageProcess(img), graph.X, graph.Y
}

func BuildText(builder *renderer.Builder, mesurement any, format string, unit string, text *theme.Text, defaultSize int) (*device.ImageProcess, int, int) {
	str := fmt.Sprintf(format, mesurement)
	if text.ShowUnit {
		str += unit
	}
	return buildText(builder, str, text, defaultSize)
}

func BuildTextFloat(builder *renderer.Builder, mesurement float64, fn func(f float64, b bool) string, text *theme.Text, defaultSize int) (*device.ImageProcess, int, int) {
	str := fn(mesurement, text.ShowUnit)
	return buildText(builder, str, text, defaultSize)
}

func BuildTextUint(builder *renderer.Builder, mesurement uint64, fn func(f uint64, b bool) string, text *theme.Text, defaultSize int) (*device.ImageProcess, int, int) {
	str := fn(mesurement, text.ShowUnit)
	return buildText(builder, str, text, defaultSize)
}

func BuildTextDt(builder *renderer.Builder, mesurement time.Time, format theme.FormatDateTime, text *theme.Text, defaultSize int) (*device.ImageProcess, int, int) {
	str := mesurement.Format(text.Format.String(format))
	return buildText(builder, str, text, defaultSize)
}

func buildText(builder *renderer.Builder, str string, text *theme.Text, defaultSize int) (*device.ImageProcess, int, int) {
	img, x, y := builder.DrawText(str, text, defaultSize)
	return device.NewImageProcess(img), x, y
}

// BuildRadialWithText renders a Radial and Text together if they overlap,
// or separately if they don't. Returns one or two payloads.
func BuildRadialWithText(builder *renderer.Builder, value float64, radial *theme.Radial, text *theme.Text, format, unit string, textSize int, p *command.UpdatePayload) []*command.UpdatePayload {
	var payloads []*command.UpdatePayload

	radialX := radial.X - radial.Radius
	radialY := radial.Y - radial.Radius
	diameter := radial.Radius * 2
	radialRect := utils.Rect{X: radialX, Y: radialY, W: diameter, H: diameter}

	str := fmt.Sprintf(format, value)
	if text.ShowUnit {
		str += unit
	}

	// Calculate text rect for overlap detection
	textRect := textCropRect(builder, str, text, textSize)

	if radialRect.Intersects(textRect) {
		// Draw both on same background and crop union
		composed, cx, cy := builder.DrawRadialAndText(value, radial, str, text, textSize)
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(composed), cx, cy))
	} else {
		// Send separately
		img := builder.DrawRadialProgressBar(value, radial)
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(img), radialX, radialY))
		textImg, textX, textY := builder.DrawText(str, text, textSize)
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(textImg), textX, textY))
	}
	return payloads
}

// BuildGraphWithText renders a Graph and Text together if they overlap,
// or separately if they don't. Returns one or two payloads.
func BuildGraphWithText(builder *renderer.Builder, value float64, graph *theme.Graph, text *theme.Text, format, unit string, textSize int, p *command.UpdatePayload) []*command.UpdatePayload {
	var payloads []*command.UpdatePayload

	graphRect := utils.Rect{X: graph.X, Y: graph.Y, W: graph.Width, H: graph.Height}

	str := fmt.Sprintf(format, value)
	if text.ShowUnit {
		str += unit
	}

	// Calculate text rect for overlap detection
	textRect := textCropRect(builder, str, text, textSize)

	if graphRect.Intersects(textRect) {
		// Draw both on same background and crop union
		composed, cx, cy := builder.DrawGraphAndText(value, graph, str, text, textSize)
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(composed), cx, cy))
	} else {
		// Send separately
		graphImg := builder.DrawProgressBar(value, graph)
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(graphImg), graph.X, graph.Y))
		textImg, textX, textY := builder.DrawText(str, text, textSize)
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(textImg), textX, textY))
	}
	return payloads
}

// textCropRect computes the crop rectangle for a text element (for overlap detection).
func textCropRect(builder *renderer.Builder, text string, stat *theme.Text, defaultSize int) utils.Rect {
	d := &font.Drawer{Face: stat.Font}

	// Crop width: larger of placeholder or actual text
	var cropWidth int
	if stat.Placeholder != "" {
		placeholderWidth := d.MeasureString(stat.Placeholder).Ceil()
		textWidth := d.MeasureString(text).Ceil()
		if placeholderWidth > textWidth {
			cropWidth = placeholderWidth
		} else {
			cropWidth = textWidth
		}
	} else {
		cropWidth = d.MeasureString(text).Ceil()
	}

	metrics := stat.Font.Metrics()
	cropHeight := (metrics.Ascent + metrics.Descent).Ceil()

	if stat.Width > 0 {
		cropWidth = stat.Width
	}
	if stat.Height > 0 {
		cropHeight = stat.Height
	}

	cropX := stat.X
	switch stat.Align {
	case theme.CENTER:
		cropX = stat.X - cropWidth/2
		if cropX < 0 {
			cropX = 0
		}
	case theme.RIGHT:
		cropX = stat.X - cropWidth
		if cropX < 0 {
			cropX = 0
		}
	}

	return utils.Rect{X: cropX, Y: stat.Y, W: cropWidth, H: cropHeight}
}

// BuildMesurement renders all active widgets in a Mesurement.
// Only ONE primary visual widget is active per measurement (priority: Chart > Gauge > Radial > StatusBar > Graph).
// Text/PercentText are complementary and can compose with the primary widget.
func BuildMesurement(builder *renderer.Builder, value float64, format, unit string, textSize int, e *theme.Mesurement, p *command.UpdatePayload, source string) []*command.UpdatePayload {
	var payloads []*command.UpdatePayload

	hasRadial := e.Radial != nil && e.Radial.Show
	hasGauge := e.Gauge != nil && e.Gauge.Show
	hasGraph := e.Graph != nil && e.Graph.Show
	hasStatusBar := e.StatusBar != nil && e.StatusBar.Show
	hasChart := e.Chart != nil && e.Chart.Show
	hasText := e.Text != nil && e.Text.Show
	hasPercent := e.Percent != nil && e.Percent.Show

	// Determine the primary visual widget (only one active)
	primaryHandled := false

	// Priority 1: Chart (time-series — standalone, no composition with text)
	if hasChart && !primaryHandled {
		e.Chart.AddSample(value)
		img := builder.DrawChart(e.Chart)
		payloads = append(payloads, p.SendPayloadFrom(device.NewImageProcess(img), e.Chart.X, e.Chart.Y, source))
		primaryHandled = true
	}

	// Priority 2: Gauge (needle — standalone)
	if hasGauge && !primaryHandled {
		img := builder.DrawGauge(value, e.Gauge)
		gx := e.Gauge.X - e.Gauge.Radius
		gy := e.Gauge.Y - e.Gauge.Radius
		payloads = append(payloads, p.SendPayloadFrom(device.NewImageProcess(img), gx, gy, source))
		primaryHandled = true
	}

	// Priority 3: Radial (arc) — can compose with Text
	if hasRadial && !primaryHandled {
		if hasText {
			payloads = append(payloads, BuildRadialWithText(builder, value, e.Radial, e.Text, format, unit, textSize, p)...)
			hasText = false
		} else if hasPercent {
			payloads = append(payloads, BuildRadialWithText(builder, value, e.Radial, e.Percent, format, unit, textSize, p)...)
			hasPercent = false
		} else {
			img, x, y := BuildRadial(builder, value, e.Radial)
			payloads = append(payloads, p.SendPayloadFrom(img, x, y, source))
		}
		primaryHandled = true
	}

	// Priority 4: StatusBar (slider with indicator — standalone)
	if hasStatusBar && !primaryHandled {
		img := builder.DrawStatusBar(value, e.StatusBar)
		ir := e.StatusBar.IndicatorRadius
		if ir <= 0 {
			ir = e.StatusBar.Height
		}
		sx := e.StatusBar.X - ir
		sy := e.StatusBar.Y - ir
		if sx < 0 {
			sx = 0
		}
		if sy < 0 {
			sy = 0
		}
		payloads = append(payloads, p.SendPayloadFrom(device.NewImageProcess(img), sx, sy, source))
		primaryHandled = true
	}

	// Priority 5: Graph (progress bar) — can compose with Text
	if hasGraph && !primaryHandled {
		if hasText {
			payloads = append(payloads, BuildGraphWithText(builder, value, e.Graph, e.Text, format, unit, textSize, p)...)
			hasText = false
		} else if hasPercent {
			payloads = append(payloads, BuildGraphWithText(builder, value, e.Graph, e.Percent, format, unit, textSize, p)...)
			hasPercent = false
		} else {
			img, x, y := BuildGraph(builder, value, e.Graph)
			payloads = append(payloads, p.SendPayloadFrom(img, x, y, source))
		}
		primaryHandled = true
	}

	// Remaining standalone Text/Percent (if not consumed by composition above)
	if hasText {
		img, x, y := BuildText(builder, value, format, unit, e.Text, textSize)
		payloads = append(payloads, p.SendPayloadFrom(img, x, y, source))
	}
	if hasPercent {
		img, x, y := BuildText(builder, value, format, unit, e.Percent, textSize)
		payloads = append(payloads, p.SendPayloadFrom(img, x, y, source))
	}

	return payloads
}

// BuildMesurementFloat is like BuildMesurement but uses a float formatting function.
func BuildMesurementFloat(builder *renderer.Builder, value float64, fn func(f float64, b bool) string, textSize int, e *theme.Mesurement, p *command.UpdatePayload, source string) []*command.UpdatePayload {
	var payloads []*command.UpdatePayload

	hasRadial := e.Radial != nil && e.Radial.Show
	hasGraph := e.Graph != nil && e.Graph.Show
	hasText := e.Text != nil && e.Text.Show

	// For float format, we generate the string respecting ShowUnit
	var str string
	if hasText && e.Text != nil {
		str = fn(value, e.Text.ShowUnit)
	} else {
		str = fn(value, true)
	}

	if hasRadial && hasText {
		// Draw both on same background copy
		radialX := e.Radial.X - e.Radial.Radius
		radialY := e.Radial.Y - e.Radial.Radius
		diameter := e.Radial.Radius * 2
		radialRect := utils.Rect{X: radialX, Y: radialY, W: diameter, H: diameter}
		textRect := textCropRect(builder, str, e.Text, textSize)

		if radialRect.Intersects(textRect) {
			composed, cx, cy := builder.DrawRadialAndText(value, e.Radial, str, e.Text, textSize)
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(composed), cx, cy))
		} else {
			img := builder.DrawRadialProgressBar(value, e.Radial)
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(img), radialX, radialY))
			textImg, textX, textY := builder.DrawText(str, e.Text, textSize)
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(textImg), textX, textY))
		}
		hasText = false
	} else if hasRadial {
		img, x, y := BuildRadial(builder, value, e.Radial)
		payloads = append(payloads, p.SendPayload(img, x, y))
	}

	if hasGraph && hasText {
		graphRect := utils.Rect{X: e.Graph.X, Y: e.Graph.Y, W: e.Graph.Width, H: e.Graph.Height}
		textRect := textCropRect(builder, str, e.Text, textSize)

		if graphRect.Intersects(textRect) {
			composed, cx, cy := builder.DrawGraphAndText(value, e.Graph, str, e.Text, textSize)
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(composed), cx, cy))
		} else {
			graphImg := builder.DrawProgressBar(value, e.Graph)
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(graphImg), e.Graph.X, e.Graph.Y))
			textImg, textX, textY := builder.DrawText(str, e.Text, textSize)
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(textImg), textX, textY))
		}
		hasText = false
	} else if hasGraph {
		img, x, y := BuildGraph(builder, value, e.Graph)
		payloads = append(payloads, p.SendPayload(img, x, y))
	}

	if hasText {
		img, x, y := BuildTextFloat(builder, value, fn, e.Text, textSize)
		payloads = append(payloads, p.SendPayload(img, x, y))
	}

	return payloads
}
