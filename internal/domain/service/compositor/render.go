package compositor

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/utils"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// drawItem represents a single widget to be drawn, with its z-index.
type drawItem struct {
	index    int
	drawFunc func(frame *image.NRGBA)
}

// renderFrame draws all active sensor widgets onto a copy of the background, respecting INDEX order.
func (c *Compositor) renderFrame(vals *SensorValues) *image.NRGBA {
	frame := imageToNRGBA(c.builder.GetBackground())
	stats := c.stats
	if stats == nil {
		return frame
	}

	// Collect all drawable items with their index
	var items []drawItem

	// CPU
	if stats.CPU != nil {
		if m := stats.CPU.Percentage; m != nil {
			items = append(items, c.collectMeasurementItems(vals.CPUPercent, "%3.0f", "%", m)...)
		}
		if m := stats.CPU.Temperature; m != nil {
			items = append(items, c.collectMeasurementItems(vals.CPUTemp, "%3.0f", "°C", m)...)
		}
		if m := stats.CPU.Frequency; m != nil {
			items = append(items, c.collectMeasurementFloatItems(vals.CPUFrequency, utils.Hertz, m)...)
		}
		if m := stats.CPU.Fan; m != nil {
			items = append(items, c.collectMeasurementItems(vals.CPUFan, "%.0f", " RPM", m)...)
		}
		if m := stats.CPU.Power; m != nil {
			items = append(items, c.collectMeasurementItems(vals.CPUPower, "%.0f", "W", m)...)
		}
		if m := stats.CPU.Voltage; m != nil {
			items = append(items, c.collectMeasurementItems(vals.CPUVoltage, "%.3f", "V", m)...)
		}
	}

	// GPU
	if stats.GPU != nil {
		if m := stats.GPU.Percentage; m != nil {
			items = append(items, c.collectMeasurementItems(vals.GPUPercent, "%3.0f", "%", m)...)
		}
		if m := stats.GPU.Temperature; m != nil {
			items = append(items, c.collectMeasurementItems(vals.GPUTemp, "%3.0f", "°C", m)...)
		}
		if m := stats.GPU.Memory; m != nil {
			items = append(items, c.collectMeasurementItems(vals.GPUMemory, "%3.1f", "%", m)...)
		}
		if m := stats.GPU.Power; m != nil {
			items = append(items, c.collectMeasurementItems(vals.GPUPower, "%.0f", "W", m)...)
		}
		if m := stats.GPU.Frequency; m != nil {
			items = append(items, c.collectMeasurementFloatItems(vals.GPUFrequency, utils.Hertz, m)...)
		}
		if m := stats.GPU.Voltage; m != nil {
			items = append(items, c.collectMeasurementItems(vals.GPUVoltage, "%4.0f", "mV", m)...)
		}
		if m := stats.GPU.Fan; m != nil {
			items = append(items, c.collectMeasurementItems(vals.GPUFan, "%.0f", " RPM", m)...)
		}
	}

	// Memory
	if stats.Memory != nil {
		if m := stats.Memory.Virtual; m != nil {
			items = append(items, c.collectMemItems(vals.MemPercent, m)...)
		}
		if m := stats.Memory.Swap; m != nil {
			items = append(items, c.collectMemItems(vals.SwapPercent, m)...)
		}
	}

	// Disk
	if stats.Disk != nil {
		if m := stats.Disk.Used; m != nil {
			items = append(items, c.collectMeasurementItems(vals.DiskPercent, "%3.0f", "%", m)...)
		}
		if m := stats.Disk.Free; m != nil {
			items = append(items, c.collectMeasurementItems(vals.DiskFree, "%3.0f", "%", m)...)
		}
		if m := stats.Disk.Temperature; m != nil {
			items = append(items, c.collectMeasurementItems(vals.DiskTemp, "%3.0f", "°C", m)...)
		}
	}

	// Network
	if stats.Net != nil {
		if stats.Net.Wired != nil {
			if u := stats.Net.Wired.Upload; u != nil && u.Text != nil && u.Text.Show {
				str := utils.NetSpeed(vals.NetUpSpeed, true)
				idx := u.Text.Index
				items = append(items, drawItem{index: idx, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, str, u.Text) }})
			}
			if d := stats.Net.Wired.Download; d != nil && d.Text != nil && d.Text.Show {
				str := utils.NetSpeed(vals.NetDownSpeed, true)
				idx := d.Text.Index
				items = append(items, drawItem{index: idx, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, str, d.Text) }})
			}
		}
	}

	// Date/Time
	if stats.Date != nil {
		if stats.Date.Hour != nil && stats.Date.Hour.Text != nil && stats.Date.Hour.Text.Show {
			str := vals.DateHour.Format(stats.Date.Hour.Text.Format.String(theme.TIME))
			idx := stats.Date.Hour.Text.Index
			items = append(items, drawItem{index: idx, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, str, stats.Date.Hour.Text) }})
		}
		if stats.Date.Day != nil && stats.Date.Day.Text != nil && stats.Date.Day.Text.Show {
			str := vals.DateDay.Format(stats.Date.Day.Text.Format.String(theme.DATE))
			idx := stats.Date.Day.Text.Index
			items = append(items, drawItem{index: idx, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, str, stats.Date.Day.Text) }})
		}
	}

	// Weather
	if stats.Weather != nil {
		if stats.Weather.Temperature != nil && stats.Weather.Temperature.Text != nil && stats.Weather.Temperature.Text.Show {
			str := fmt.Sprintf("%.0f°C", vals.WeatherTemp)
			idx := stats.Weather.Temperature.Text.Index
			items = append(items, drawItem{index: idx, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, str, stats.Weather.Temperature.Text) }})
		}
		if stats.Weather.Condition != nil && stats.Weather.Condition.Show {
			str := fmt.Sprintf("%s, %.0f°C, Wind %.0fkm/h", vals.WeatherDesc, vals.WeatherTemp, vals.WeatherWind)
			idx := stats.Weather.Condition.Index
			items = append(items, drawItem{index: idx, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, str, stats.Weather.Condition) }})
		}
	}

	// Volume
	if stats.Volume != nil {
		if stats.Volume.Text != nil && stats.Volume.Text.Show {
			str := fmt.Sprintf("%.0f", vals.Volume)
			idx := stats.Volume.Text.Index
			items = append(items, drawItem{index: idx, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, str, stats.Volume.Text) }})
		}
	}

	// Sort by INDEX (lower index = drawn first = behind)
	sort.Slice(items, func(i, j int) bool {
		return items[i].index < items[j].index
	})

	// Draw all in order
	for _, item := range items {
		item.drawFunc(frame)
	}

	return frame
}

// collectMeasurementItems returns draw items for all active widgets in a Measurement.
func (c *Compositor) collectMeasurementItems(value float64, format, unit string, m *theme.Mesurement) []drawItem {
	var items []drawItem
	str := fmt.Sprintf(format, value)

	if m.Chart != nil && m.Chart.Show {
		m.Chart.AddSample(value)
		chart := m.Chart
		items = append(items, drawItem{index: chart.Index, drawFunc: func(f *image.NRGBA) { c.drawChartOnFrame(f, chart) }})
	} else if m.Gauge != nil && m.Gauge.Show {
		g := m.Gauge
		items = append(items, drawItem{index: g.Index, drawFunc: func(f *image.NRGBA) { c.drawGaugeOnFrame(f, value, g) }})
	} else if m.Radial != nil && m.Radial.Show {
		r := m.Radial
		items = append(items, drawItem{index: r.Index, drawFunc: func(f *image.NRGBA) { c.drawRadialOnFrame(f, value, r) }})
	} else if m.StatusBar != nil && m.StatusBar.Show {
		s := m.StatusBar
		items = append(items, drawItem{index: s.Index, drawFunc: func(f *image.NRGBA) { c.drawStatusBarOnFrame(f, value, s) }})
	} else if m.Graph != nil && m.Graph.Show {
		g := m.Graph
		items = append(items, drawItem{index: g.Index, drawFunc: func(f *image.NRGBA) { c.drawGraphOnFrame(f, value, g) }})
	}

	if m.Text != nil && m.Text.Show {
		t := m.Text
		s := str
		if t.ShowUnit {
			s += unit
		}
		items = append(items, drawItem{index: t.Index, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, s, t) }})
	}
	if m.Percent != nil && m.Percent.Show {
		t := m.Percent
		s := str
		if t.ShowUnit {
			s += unit
		}
		items = append(items, drawItem{index: t.Index, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, s, t) }})
	}

	return items
}

// collectMeasurementFloatItems is like collectMeasurementItems but uses a float formatter.
func (c *Compositor) collectMeasurementFloatItems(value float64, fn func(float64, bool) string, m *theme.Mesurement) []drawItem {
	var items []drawItem

	if m.Chart != nil && m.Chart.Show {
		m.Chart.AddSample(value)
		chart := m.Chart
		items = append(items, drawItem{index: chart.Index, drawFunc: func(f *image.NRGBA) { c.drawChartOnFrame(f, chart) }})
	} else if m.Radial != nil && m.Radial.Show {
		r := m.Radial
		items = append(items, drawItem{index: r.Index, drawFunc: func(f *image.NRGBA) { c.drawRadialOnFrame(f, value, r) }})
	} else if m.Graph != nil && m.Graph.Show {
		g := m.Graph
		items = append(items, drawItem{index: g.Index, drawFunc: func(f *image.NRGBA) { c.drawGraphOnFrame(f, value, g) }})
	}

	if m.Text != nil && m.Text.Show {
		t := m.Text
		s := fn(value, t.ShowUnit)
		items = append(items, drawItem{index: t.Index, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, s, t) }})
	}

	return items
}

// collectMemItems returns draw items for a MemMesurement.
func (c *Compositor) collectMemItems(percent float64, m *theme.MemMesurement) []drawItem {
	var items []drawItem

	if m.Radial != nil && m.Radial.Show {
		r := m.Radial
		items = append(items, drawItem{index: r.Index, drawFunc: func(f *image.NRGBA) { c.drawRadialOnFrame(f, percent, r) }})
	}
	if m.Graph != nil && m.Graph.Show {
		g := m.Graph
		items = append(items, drawItem{index: g.Index, drawFunc: func(f *image.NRGBA) { c.drawGraphOnFrame(f, percent, g) }})
	}
	if m.PercentText != nil && m.PercentText.Show {
		t := m.PercentText
		s := fmt.Sprintf("%3.0f%%", percent)
		items = append(items, drawItem{index: t.Index, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, s, t) }})
	}

	return items
}

// --- Drawing functions (draw directly on frame, no crop) ---

func (c *Compositor) drawTextOnFrame(frame *image.NRGBA, text string, stat *theme.Text) {
	if stat.Font == nil {
		return
	}

	metrics := stat.Font.Metrics()
	textWidth := font.MeasureString(stat.Font, text)

	// Calculate X based on alignment
	var dotX fixed.Int26_6
	switch stat.Align {
	case theme.CENTER:
		dotX = fixed.I(stat.X) - textWidth/2
	case theme.RIGHT:
		dotX = fixed.I(stat.X) - textWidth
	default: // LEFT
		dotX = fixed.I(stat.X)
	}

	dotY := fixed.I(stat.Y) + metrics.Ascent

	d := &font.Drawer{
		Dst:  frame,
		Src:  image.NewUniform(stat.FontColor),
		Face: stat.Font,
		Dot:  fixed.Point26_6{X: dotX, Y: dotY},
	}
	d.DrawString(text)
}

func (c *Compositor) drawGraphOnFrame(frame *image.NRGBA, value float64, stat *theme.Graph) {
	ratio := value / float64(stat.MaxValue-stat.MinValue)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}

	direction := strings.ToLower(stat.Direction)
	if direction == "" {
		direction = "left"
	}

	var filledSize int
	switch direction {
	case "left", "right":
		filledSize = int(math.Round(ratio * float64(stat.Width)))
	case "up", "down":
		filledSize = int(math.Round(ratio * float64(stat.Height)))
	}

	if stat.Steps > 0 && stat.StepGap > 0 {
		c.drawSegmentedBarOnFrame(frame, stat, direction, filledSize)
	} else {
		switch direction {
		case "left":
			fillNRGBA(frame, stat.X, stat.Y, filledSize, stat.Height, stat.BarColor)
		case "right":
			fillNRGBA(frame, stat.X+stat.Width-filledSize, stat.Y, filledSize, stat.Height, stat.BarColor)
		case "up":
			fillNRGBA(frame, stat.X, stat.Y+stat.Height-filledSize, stat.Width, filledSize, stat.BarColor)
		case "down":
			fillNRGBA(frame, stat.X, stat.Y, stat.Width, filledSize, stat.BarColor)
		}
	}

	if stat.BarOutline {
		strokeNRGBA(frame, stat.X, stat.Y, stat.Width, stat.Height, stat.BarColor)
	}
}

func (c *Compositor) drawSegmentedBarOnFrame(frame *image.NRGBA, stat *theme.Graph, direction string, filledSize int) {
	steps := stat.Steps
	gap := stat.StepGap

	switch direction {
	case "left", "right":
		segW := (stat.Width - (steps-1)*gap) / steps
		if segW < 1 {
			segW = 1
		}
		filledSteps := int(math.Round(float64(filledSize) / float64(stat.Width) * float64(steps)))
		for i := 0; i < filledSteps && i < steps; i++ {
			var sx int
			if direction == "left" {
				sx = stat.X + i*(segW+gap)
			} else {
				sx = stat.X + stat.Width - (i+1)*(segW+gap) + gap
			}
			fillNRGBA(frame, sx, stat.Y, segW, stat.Height, stat.BarColor)
		}
	case "up", "down":
		segH := (stat.Height - (steps-1)*gap) / steps
		if segH < 1 {
			segH = 1
		}
		filledSteps := int(math.Round(float64(filledSize) / float64(stat.Height) * float64(steps)))
		for i := 0; i < filledSteps && i < steps; i++ {
			var sy int
			if direction == "up" {
				sy = stat.Y + stat.Height - (i+1)*(segH+gap) + gap
			} else {
				sy = stat.Y + i*(segH+gap)
			}
			fillNRGBA(frame, stat.X, sy, stat.Width, segH, stat.BarColor)
		}
	}
}

func (c *Compositor) drawRadialOnFrame(frame *image.NRGBA, value float64, stat *theme.Radial) {
	x, y := float64(stat.X), float64(stat.Y)
	amin := utils.Radians(stat.AngleStart)
	amax := utils.Radians(180 + stat.AngleStart + stat.AngleEnd)
	total := (value * float64(180+stat.AngleStart+stat.AngleEnd)) / 100
	cur := utils.Radians(int(total) + stat.AngleStart)
	if cur > amax {
		cur = amax
	}

	arcRadius := float64(stat.Radius - (stat.Width / 2))

	if stat.AngleSteps > 1 && stat.AngleSep > 0 {
		totalAngle := cur - amin
		segAngle := totalAngle / float64(stat.AngleSteps)
		sepAngle := utils.Radians(stat.AngleSep)
		for i := 0; i < stat.AngleSteps; i++ {
			segStart := amin + float64(i)*segAngle
			segEnd := segStart + segAngle - sepAngle
			if segEnd > cur {
				segEnd = cur
			}
			if segEnd > segStart {
				drawArcNRGBA(frame, x, y, arcRadius, segStart, segEnd, stat.Width, stat.BarColor)
			}
		}
	} else {
		drawArcNRGBA(frame, x, y, arcRadius, amin, cur, stat.Width, stat.BarColor)
	}

	if stat.ShowText && stat.Font != nil {
		measure := fmt.Sprintf("%3.0f", value)
		if stat.ShowUnit {
			measure += "%"
		}
		tw := font.MeasureString(stat.Font, measure)
		metrics := stat.Font.Metrics()
		dotX := fixed.I(stat.X) - tw/2
		dotY := fixed.I(stat.Y) + metrics.Ascent/2 - metrics.Descent/2
		d := &font.Drawer{
			Dst:  frame,
			Src:  image.NewUniform(stat.FontColor),
			Face: stat.Font,
			Dot:  fixed.Point26_6{X: dotX, Y: dotY},
		}
		d.DrawString(measure)
	}
}

func (c *Compositor) drawGaugeOnFrame(frame *image.NRGBA, value float64, stat *theme.Gauge) {
	x, y := float64(stat.X), float64(stat.Y)
	ratio := value / float64(stat.MaxValue-stat.MinValue)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	totalAngle := float64(180 + stat.AngleStart + stat.AngleEnd)
	needleAngle := utils.Radians(stat.AngleStart) + ratio*utils.Radians(int(totalAngle))

	needleLen := float64(stat.Radius) * 0.85
	width := stat.NeedleWidth
	if width <= 0 {
		width = 2
	}

	endX := x + needleLen*math.Cos(needleAngle)
	endY := y + needleLen*math.Sin(needleAngle)
	drawLineNRGBA(frame, x, y, endX, endY, width, stat.NeedleColor)
	drawCircleNRGBA(frame, x, y, float64(width+1), stat.NeedleColor)

	if stat.ShowText && stat.Font != nil {
		measure := fmt.Sprintf("%3.0f", value)
		if stat.ShowUnit {
			measure += "%"
		}
		tw := font.MeasureString(stat.Font, measure)
		metrics := stat.Font.Metrics()
		dotX := fixed.I(stat.X) - tw/2
		dotY := fixed.I(stat.Y) + metrics.Ascent/2 - metrics.Descent/2 + fixed.I(int(float64(stat.Radius)*0.35))
		d := &font.Drawer{
			Dst:  frame,
			Src:  image.NewUniform(stat.FontColor),
			Face: stat.Font,
			Dot:  fixed.Point26_6{X: dotX, Y: dotY},
		}
		d.DrawString(measure)
	}
}

func (c *Compositor) drawStatusBarOnFrame(frame *image.NRGBA, value float64, stat *theme.StatusBar) {
	ratio := value / float64(stat.MaxValue-stat.MinValue)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	filledWidth := int(math.Round(ratio * float64(stat.Width)))
	fillNRGBA(frame, stat.X, stat.Y, filledWidth, stat.Height, stat.BarColor)

	// Indicator circle
	ir := stat.IndicatorRadius
	if ir <= 0 {
		ir = stat.Height
	}
	cx := float64(stat.X + filledWidth)
	cy := float64(stat.Y + stat.Height/2)
	drawCircleNRGBA(frame, cx, cy, float64(ir), stat.IndicatorColor)
}

func (c *Compositor) drawChartOnFrame(frame *image.NRGBA, stat *theme.Chart) {
	samples := stat.GetSamples()
	if len(samples) == 0 {
		return
	}

	valueRange := float64(stat.MaxValue - stat.MinValue)
	if valueRange <= 0 {
		valueRange = 100
	}

	style := strings.ToLower(stat.Style)

	if style == "line" && len(samples) >= 2 {
		spacing := float64(stat.Width) / float64(len(samples)-1)
		for i := 0; i < len(samples)-1; i++ {
			r1 := math.Min(1, math.Max(0, samples[i]/valueRange))
			r2 := math.Min(1, math.Max(0, samples[i+1]/valueRange))
			x1 := float64(stat.X) + float64(i)*spacing
			x2 := float64(stat.X) + float64(i+1)*spacing
			y1 := float64(stat.Y+stat.Height) - r1*float64(stat.Height)
			y2 := float64(stat.Y+stat.Height) - r2*float64(stat.Height)
			drawLineNRGBA(frame, x1, y1, x2, y2, 1, stat.LineColor)
		}
	} else {
		colStep := stat.ColumnWidth + stat.ColumnGap
		if colStep <= 0 {
			colStep = 2
		}
		for i := len(samples) - 1; i >= 0; i-- {
			colIndex := len(samples) - 1 - i
			colX := stat.X + stat.Width - (colIndex+1)*colStep
			if colX < stat.X {
				break
			}
			ratio := math.Min(1, math.Max(0, samples[i]/valueRange))
			barH := int(math.Round(ratio * float64(stat.Height)))
			barY := stat.Y + stat.Height - barH
			fillNRGBA(frame, colX, barY, stat.ColumnWidth, barH, stat.FillColor)
		}
	}

	if stat.BorderWidth > 0 {
		strokeNRGBA(frame, stat.X, stat.Y, stat.Width, stat.Height, stat.LineColor)
	}
}

// --- Primitive drawing functions on *image.NRGBA ---

func fillNRGBA(dst *image.NRGBA, x, y, w, h int, c color.Color) {
	r, g, b, a := c.RGBA()
	nc := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	bounds := dst.Bounds()
	for py := y; py < y+h && py < bounds.Max.Y; py++ {
		for px := x; px < x+w && px < bounds.Max.X; px++ {
			if px >= 0 && py >= 0 {
				dst.SetNRGBA(px, py, nc)
			}
		}
	}
}

func strokeNRGBA(dst *image.NRGBA, x, y, w, h int, c color.Color) {
	r, g, b, a := c.RGBA()
	nc := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	for px := x; px < x+w; px++ {
		if px >= 0 && px < dst.Bounds().Max.X {
			if y >= 0 {
				dst.SetNRGBA(px, y, nc)
			}
			if y+h-1 < dst.Bounds().Max.Y {
				dst.SetNRGBA(px, y+h-1, nc)
			}
		}
	}
	for py := y; py < y+h; py++ {
		if py >= 0 && py < dst.Bounds().Max.Y {
			if x >= 0 {
				dst.SetNRGBA(x, py, nc)
			}
			if x+w-1 < dst.Bounds().Max.X {
				dst.SetNRGBA(x+w-1, py, nc)
			}
		}
	}
}

func drawArcNRGBA(dst *image.NRGBA, cx, cy, radius, startAngle, endAngle float64, width int, c color.Color) {
	r, g, b, a := c.RGBA()
	nc := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	halfW := float64(width) / 2.0
	innerR := radius - halfW
	outerR := radius + halfW
	if innerR < 0 {
		innerR = 0
	}

	minX := int(math.Floor(cx - outerR - 1))
	maxX := int(math.Ceil(cx + outerR + 1))
	minY := int(math.Floor(cy - outerR - 1))
	maxY := int(math.Ceil(cy + outerR + 1))

	bounds := dst.Bounds()
	if minX < bounds.Min.X {
		minX = bounds.Min.X
	}
	if minY < bounds.Min.Y {
		minY = bounds.Min.Y
	}
	if maxX > bounds.Max.X {
		maxX = bounds.Max.X
	}
	if maxY > bounds.Max.Y {
		maxY = bounds.Max.Y
	}

	twoPi := 2 * math.Pi
	for py := minY; py < maxY; py++ {
		for px := minX; px < maxX; px++ {
			dx := float64(px) + 0.5 - cx
			dy := float64(py) + 0.5 - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < innerR || dist > outerR {
				continue
			}
			angle := math.Mod(math.Atan2(dy, dx)+twoPi, twoPi)
			s := math.Mod(startAngle+twoPi, twoPi)
			e := math.Mod(endAngle+twoPi, twoPi)
			if s <= e {
				if angle >= s && angle <= e {
					dst.SetNRGBA(px, py, nc)
				}
			} else {
				if angle >= s || angle <= e {
					dst.SetNRGBA(px, py, nc)
				}
			}
		}
	}
}

func drawLineNRGBA(dst *image.NRGBA, x1, y1, x2, y2 float64, width int, c color.Color) {
	r, g, b, a := c.RGBA()
	nc := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	dx := x2 - x1
	dy := y2 - y1
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return
	}
	halfW := float64(width) / 2.0
	steps := int(length * 2)
	if steps < 10 {
		steps = 10
	}
	bounds := dst.Bounds()
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		px := x1 + dx*t
		py := y1 + dy*t
		for wy := -halfW; wy <= halfW; wy++ {
			for wx := -halfW; wx <= halfW; wx++ {
				if wx*wx+wy*wy <= halfW*halfW {
					ix := int(math.Round(px + wx))
					iy := int(math.Round(py + wy))
					if ix >= 0 && iy >= 0 && ix < bounds.Max.X && iy < bounds.Max.Y {
						dst.SetNRGBA(ix, iy, nc)
					}
				}
			}
		}
	}
}

func drawCircleNRGBA(dst *image.NRGBA, cx, cy, radius float64, c color.Color) {
	r, g, b, a := c.RGBA()
	nc := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	rSq := radius * radius
	bounds := dst.Bounds()
	for py := int(cy - radius); py <= int(cy+radius); py++ {
		for px := int(cx - radius); px <= int(cx+radius); px++ {
			ddx := float64(px) + 0.5 - cx
			ddy := float64(py) + 0.5 - cy
			if ddx*ddx+ddy*ddy <= rSq {
				if px >= 0 && py >= 0 && px < bounds.Max.X && py < bounds.Max.Y {
					dst.SetNRGBA(px, py, nc)
				}
			}
		}
	}
}
