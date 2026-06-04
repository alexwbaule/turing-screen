package sensors

import (
	"fmt"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
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

	radialImg := builder.DrawRadialProgressBar(value, radial)
	radialX := radial.X - radial.Radius
	radialY := radial.Y - radial.Radius
	radialRect := utils.Rect{X: radialX, Y: radialY, W: radialImg.Bounds().Dx(), H: radialImg.Bounds().Dy()}

	str := fmt.Sprintf(format, value)
	if text.ShowUnit {
		str += unit
	}
	textImg, textX, textY := builder.DrawText(str, text, textSize)
	textRect := utils.Rect{X: textX, Y: textY, W: textImg.Bounds().Dx(), H: textImg.Bounds().Dy()}

	if radialRect.Intersects(textRect) {
		// Compose into one image
		composed, cx, cy := utils.ComposeInUnion(radialImg, radialRect, textImg, textRect)
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(composed), cx, cy))
	} else {
		// Send separately
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(radialImg), radialX, radialY))
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(textImg), textX, textY))
	}
	return payloads
}

// BuildGraphWithText renders a Graph and Text together if they overlap,
// or separately if they don't. Returns one or two payloads.
func BuildGraphWithText(builder *renderer.Builder, value float64, graph *theme.Graph, text *theme.Text, format, unit string, textSize int, p *command.UpdatePayload) []*command.UpdatePayload {
	var payloads []*command.UpdatePayload

	graphImg := builder.DrawProgressBar(value, graph)
	graphX := graph.X
	graphY := graph.Y
	graphRect := utils.Rect{X: graphX, Y: graphY, W: graphImg.Bounds().Dx(), H: graphImg.Bounds().Dy()}

	str := fmt.Sprintf(format, value)
	if text.ShowUnit {
		str += unit
	}
	textImg, textX, textY := builder.DrawText(str, text, textSize)
	textRect := utils.Rect{X: textX, Y: textY, W: textImg.Bounds().Dx(), H: textImg.Bounds().Dy()}

	if graphRect.Intersects(textRect) {
		composed, cx, cy := utils.ComposeInUnion(graphImg, graphRect, textImg, textRect)
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(composed), cx, cy))
	} else {
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(graphImg), graphX, graphY))
		payloads = append(payloads, p.SendPayload(device.NewImageProcess(textImg), textX, textY))
	}
	return payloads
}

// BuildMesurement renders all active widgets in a Mesurement (Radial, Graph, Text, Percent)
// with proper composition when they overlap. Returns all payloads to send.
func BuildMesurement(builder *renderer.Builder, value float64, format, unit string, textSize int, e *theme.Mesurement, p *command.UpdatePayload) []*command.UpdatePayload {
	var payloads []*command.UpdatePayload

	hasRadial := e.Radial != nil && e.Radial.Show
	hasGraph := e.Graph != nil && e.Graph.Show
	hasText := e.Text != nil && e.Text.Show
	hasPercent := e.Percent != nil && e.Percent.Show

	// Compose Radial + Text/Percent if both active
	if hasRadial && hasText {
		payloads = append(payloads, BuildRadialWithText(builder, value, e.Radial, e.Text, format, unit, textSize, p)...)
		hasText = false // already handled
	} else if hasRadial && hasPercent {
		payloads = append(payloads, BuildRadialWithText(builder, value, e.Radial, e.Percent, format, unit, textSize, p)...)
		hasPercent = false
	} else if hasRadial {
		img, x, y := BuildRadial(builder, value, e.Radial)
		payloads = append(payloads, p.SendPayload(img, x, y))
	}

	// Compose Graph + Text/Percent if both active
	if hasGraph && hasText {
		payloads = append(payloads, BuildGraphWithText(builder, value, e.Graph, e.Text, format, unit, textSize, p)...)
		hasText = false
	} else if hasGraph && hasPercent {
		payloads = append(payloads, BuildGraphWithText(builder, value, e.Graph, e.Percent, format, unit, textSize, p)...)
		hasPercent = false
	} else if hasGraph {
		img, x, y := BuildGraph(builder, value, e.Graph)
		payloads = append(payloads, p.SendPayload(img, x, y))
	}

	// Remaining standalone Text/Percent
	if hasText {
		img, x, y := BuildText(builder, value, format, unit, e.Text, textSize)
		payloads = append(payloads, p.SendPayload(img, x, y))
	}
	if hasPercent {
		img, x, y := BuildText(builder, value, format, unit, e.Percent, textSize)
		payloads = append(payloads, p.SendPayload(img, x, y))
	}

	return payloads
}

// BuildMesurementFloat is like BuildMesurement but uses a float formatting function.
func BuildMesurementFloat(builder *renderer.Builder, value float64, fn func(f float64, b bool) string, textSize int, e *theme.Mesurement, p *command.UpdatePayload) []*command.UpdatePayload {
	var payloads []*command.UpdatePayload

	hasRadial := e.Radial != nil && e.Radial.Show
	hasGraph := e.Graph != nil && e.Graph.Show
	hasText := e.Text != nil && e.Text.Show

	// For float format, we generate the string first
	str := fn(value, true)
	strNoUnit := fn(value, false)

	if hasRadial && hasText {
		// Use raw format for compose
		textImg, textX, textY := builder.DrawText(str, e.Text, textSize)
		radialImg := builder.DrawRadialProgressBar(value, e.Radial)
		radialX := e.Radial.X - e.Radial.Radius
		radialY := e.Radial.Y - e.Radial.Radius
		radialRect := utils.Rect{X: radialX, Y: radialY, W: radialImg.Bounds().Dx(), H: radialImg.Bounds().Dy()}
		textRect := utils.Rect{X: textX, Y: textY, W: textImg.Bounds().Dx(), H: textImg.Bounds().Dy()}
		if radialRect.Intersects(textRect) {
			composed, cx, cy := utils.ComposeInUnion(radialImg, radialRect, textImg, textRect)
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(composed), cx, cy))
		} else {
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(radialImg), radialX, radialY))
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(textImg), textX, textY))
		}
		hasText = false
	} else if hasRadial {
		img, x, y := BuildRadial(builder, value, e.Radial)
		payloads = append(payloads, p.SendPayload(img, x, y))
	}

	if hasGraph && hasText {
		graphImg := builder.DrawProgressBar(value, e.Graph)
		graphRect := utils.Rect{X: e.Graph.X, Y: e.Graph.Y, W: graphImg.Bounds().Dx(), H: graphImg.Bounds().Dy()}
		textImg, textX, textY := builder.DrawText(str, e.Text, textSize)
		textRect := utils.Rect{X: textX, Y: textY, W: textImg.Bounds().Dx(), H: textImg.Bounds().Dy()}
		if graphRect.Intersects(textRect) {
			composed, cx, cy := utils.ComposeInUnion(graphImg, graphRect, textImg, textRect)
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(composed), cx, cy))
		} else {
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(graphImg), e.Graph.X, e.Graph.Y))
			payloads = append(payloads, p.SendPayload(device.NewImageProcess(textImg), textX, textY))
		}
		hasText = false
	} else if hasGraph {
		img, x, y := BuildGraph(builder, value, e.Graph)
		payloads = append(payloads, p.SendPayload(img, x, y))
	}

	if hasText {
		_ = strNoUnit
		img, x, y := BuildTextFloat(builder, value, fn, e.Text, textSize)
		payloads = append(payloads, p.SendPayload(img, x, y))
	}

	return payloads
}
