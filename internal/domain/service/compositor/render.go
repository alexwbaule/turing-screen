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

// renderFrame draws all active sensor widgets onto frame (pre-filled with background), respecting INDEX order.
func (c *Compositor) renderFrame(frame *image.NRGBA, vals *sensorData) *image.NRGBA {
	stats := c.stats
	if stats == nil {
		return frame
	}

	// Collect all drawable items with their index
	var items []drawItem

	// CPU
	if stats.CPU != nil {
		if m := stats.CPU.Model; m != nil && c.models.CPU != "" {
			items = append(items, c.collectModelItem(c.models.CPU, m)...)
		}
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
		if m := stats.GPU.Model; m != nil && c.models.GPU != "" {
			items = append(items, c.collectModelItem(c.models.GPU, m)...)
		}
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
		if ram := stats.Memory.RAM; ram != nil {
			// SIZE uses the hwinfo MemTotal string; MODEL has no hwinfo source yet.
			if ram.Size != nil && c.models.Mem != "" {
				items = append(items, c.collectModelItem(c.models.Mem, ram.Size)...)
			}
			// Promote RAM usage fields into a temporary Sensor so
			// collectMeasurementItems can be reused without duplication.
			usage := &theme.Sensor{
				Graph:       ram.Graph,
				Radial:      ram.Radial,
				Gauge:       ram.Gauge,
				StatusBar:   ram.StatusBar,
				Chart:       ram.Chart,
				Text:        ram.Text,
				PercentText: ram.PercentText,
			}
			items = append(items, c.collectMeasurementItems(vals.MemPercent, "%3.0f", "%", usage)...)
		}
		if m := stats.Memory.Swap; m != nil {
			items = append(items, c.collectMeasurementItems(vals.SwapPercent, "%3.0f", "%", m)...)
		}
	}

	// Disk
	if stats.Disk != nil {
		if m := stats.Disk.Model; m != nil && c.models.Disk != "" {
			items = append(items, c.collectModelItem(c.models.Disk, m)...)
		}
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

	// Host (hostname + system load average)
	if stats.Host != nil {
		if m := stats.Host.Hostname; m != nil && c.models.Hostname != "" {
			items = append(items, c.collectModelItem(c.models.Hostname, m)...)
		}
		if load := stats.Host.Load; load != nil {
			if m := load.One; m != nil {
				items = append(items, c.collectMeasurementItems(vals.CPULoad1, "%.2f", "", m)...)
			}
			if m := load.Five; m != nil {
				items = append(items, c.collectMeasurementItems(vals.CPULoad5, "%.2f", "", m)...)
			}
			if m := load.Fifteen; m != nil {
				items = append(items, c.collectMeasurementItems(vals.CPULoad15, "%.2f", "", m)...)
			}
		}
	}

	// Network
	if stats.Net != nil {
		if stats.Net.Wired != nil {
			if m := stats.Net.Wired.Upload; m != nil {
				items = append(items, c.collectMeasurementFloatItems(vals.NetUpSpeed, utils.NetSpeed, m)...)
			}
			if m := stats.Net.Wired.Download; m != nil {
				items = append(items, c.collectMeasurementFloatItems(vals.NetDownSpeed, utils.NetSpeed, m)...)
			}
			if m := stats.Net.Wired.Uploaded; m != nil {
				v := float64(vals.NetUploaded)
				items = append(items, c.collectMeasurementFloatItems(v, func(x float64, u bool) string { return utils.Bytes(uint64(x), u) }, m)...)
			}
			if m := stats.Net.Wired.Downloaded; m != nil {
				v := float64(vals.NetDownloaded)
				items = append(items, c.collectMeasurementFloatItems(v, func(x float64, u bool) string { return utils.Bytes(uint64(x), u) }, m)...)
			}
		}
		if stats.Net.Wifi != nil {
			if m := stats.Net.Wifi.Upload; m != nil {
				items = append(items, c.collectMeasurementFloatItems(vals.WifiUpSpeed, utils.NetSpeed, m)...)
			}
			if m := stats.Net.Wifi.Download; m != nil {
				items = append(items, c.collectMeasurementFloatItems(vals.WifiDownSpeed, utils.NetSpeed, m)...)
			}
		}
	}

	// Date/Time — TEXT uses formatted string; visual types use numeric value
	if stats.Date != nil {
		if m := stats.Date.Hour; m != nil {
			hourVal := float64(vals.DateHour.Hour()) + float64(vals.DateHour.Minute())/60.0
			var hourStr string
			if m.Text != nil {
				hourStr = vals.DateHour.Format(m.Text.Format.String(theme.TIME))
			} else {
				hourStr = vals.DateHour.Format("15:04:05")
			}
			items = append(items, c.collectDateTimeItems(hourVal, hourStr, m)...)
		}
		if m := stats.Date.Day; m != nil {
			dayVal := float64(vals.DateDay.Day())
			var dayStr string
			if m.Text != nil {
				dayStr = vals.DateDay.Format(m.Text.Format.String(theme.DATE))
			} else {
				dayStr = vals.DateDay.Format("2006-01-02")
			}
			items = append(items, c.collectDateTimeItems(dayVal, dayStr, m)...)
		}
	}

	// Weather
	if stats.Weather != nil {
		if m := stats.Weather.Temperature; m != nil {
			items = append(items, c.collectMeasurementItems(vals.WeatherTemp, "%.0f", "°C", m)...)
		}
		if stats.Weather.Condition != nil && stats.Weather.Condition.Show {
			str := fmt.Sprintf("%s, %.0f°C, Wind %.0fkm/h", vals.WeatherDesc, vals.WeatherTemp, vals.WeatherWind)
			idx := stats.Weather.Condition.Index
			items = append(items, drawItem{index: idx, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, str, stats.Weather.Condition) }})
		}
	}

	// Volume (0–100 percentage)
	if m := stats.Volume; m != nil {
		items = append(items, c.collectMeasurementItems(vals.Volume, "%.0f", "%", m)...)
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

// collectMeasurementItems returns draw items for all active display types in a Sensor.
func (c *Compositor) collectMeasurementItems(value float64, format, unit string, m *theme.Sensor) []drawItem {
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
	if m.PercentText != nil && m.PercentText.Show {
		t := m.PercentText
		s := str
		if t.ShowUnit {
			s += unit
		}
		items = append(items, drawItem{index: t.Index, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, s, t) }})
	}

	return items
}

// collectMeasurementFloatItems is like collectMeasurementItems but uses a custom text formatter (e.g. NetSpeed, Hertz, Bytes).
// Visual types (Graph, Radial, Gauge, StatusBar, Chart) receive the raw float value and use MIN/MAX_VALUE from config.
func (c *Compositor) collectMeasurementFloatItems(value float64, fn func(float64, bool) string, m *theme.Sensor) []drawItem {
	var items []drawItem

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
		s := fn(value, t.ShowUnit)
		items = append(items, drawItem{index: t.Index, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, s, t) }})
	}
	if m.PercentText != nil && m.PercentText.Show {
		t := m.PercentText
		s := fn(value, t.ShowUnit)
		items = append(items, drawItem{index: t.Index, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, s, t) }})
	}

	return items
}

// collectDateTimeItems handles DateTime sensors: TEXT uses a pre-formatted string while
// visual types (Graph, Radial, Gauge, etc.) use numericVal (e.g. hour 0–23, day 1–31).
func (c *Compositor) collectDateTimeItems(numericVal float64, textStr string, m *theme.Sensor) []drawItem {
	var items []drawItem

	if m.Chart != nil && m.Chart.Show {
		m.Chart.AddSample(numericVal)
		chart := m.Chart
		items = append(items, drawItem{index: chart.Index, drawFunc: func(f *image.NRGBA) { c.drawChartOnFrame(f, chart) }})
	} else if m.Gauge != nil && m.Gauge.Show {
		g := m.Gauge
		items = append(items, drawItem{index: g.Index, drawFunc: func(f *image.NRGBA) { c.drawGaugeOnFrame(f, numericVal, g) }})
	} else if m.Radial != nil && m.Radial.Show {
		r := m.Radial
		items = append(items, drawItem{index: r.Index, drawFunc: func(f *image.NRGBA) { c.drawRadialOnFrame(f, numericVal, r) }})
	} else if m.StatusBar != nil && m.StatusBar.Show {
		s := m.StatusBar
		items = append(items, drawItem{index: s.Index, drawFunc: func(f *image.NRGBA) { c.drawStatusBarOnFrame(f, numericVal, s) }})
	} else if m.Graph != nil && m.Graph.Show {
		g := m.Graph
		items = append(items, drawItem{index: g.Index, drawFunc: func(f *image.NRGBA) { c.drawGraphOnFrame(f, numericVal, g) }})
	}

	if m.Text != nil && m.Text.Show {
		t := m.Text
		items = append(items, drawItem{index: t.Index, drawFunc: func(f *image.NRGBA) { c.drawTextOnFrame(f, textStr, t) }})
	}

	return items
}

// collectModelItem returns a draw item for a MODEL/TOTAL sensor whose value is a
// static hardware string (CPU model name, GPU model name, RAM total, disk model).
// Unlike collectMeasurementItems the value never changes between frames, but it
// still needs to be z-ordered with the other widgets by its TEXT.Index.
func (c *Compositor) collectModelItem(value string, m *theme.Sensor) []drawItem {
	var items []drawItem
	if m.Text != nil && m.Text.Show {
		t := m.Text
		s := value
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
	if stat.RevertValue {
		ratio = 1 - ratio
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

	hasGradient := stat.GradientColor != nil
	hasRound := stat.CornerRadius > 0

	fillBar := func(x, y, w, h int, clr color.Color) {
		if w <= 0 || h <= 0 || clr == nil {
			return
		}
		if hasGradient && hasRound {
			fillGradientRoundedNRGBA(frame, x, y, w, h, stat.CornerRadius, stat.GradientColor, clr)
		} else if hasGradient {
			fillGradientNRGBA(frame, x, y, w, h, stat.GradientColor, clr)
		} else if hasRound {
			fillRoundedNRGBA(frame, x, y, w, h, stat.CornerRadius, clr)
		} else {
			fillNRGBA(frame, x, y, w, h, clr)
		}
	}

	// Background track color: draw only the unfilled portion (bounding box stays transparent).
	var bgClr color.Color
	if c := stat.BackgroundStyle.BackgroundColor; c != nil {
		if _, _, _, a := c.RGBA(); a > 0 {
			bgClr = c
		}
	}

	// Compute effective steps (BlockWidth takes priority over Steps)
	steps := stat.Steps
	if stat.BlockWidth > 0 {
		gap := stat.StepGap
		if gap < 0 {
			gap = 0
		}
		steps = stat.Width / (stat.BlockWidth + gap)
		if steps < 1 {
			steps = 1
		}
	}

	if steps > 0 && stat.StepGap > 0 {
		c.drawSegmentedBarOnFrame(frame, stat, direction, filledSize, steps, fillBar, bgClr)
	} else {
		switch direction {
		case "left":
			fillBar(stat.X+filledSize, stat.Y, stat.Width-filledSize, stat.Height, bgClr)
			fillBar(stat.X, stat.Y, filledSize, stat.Height, stat.BarColor)
		case "right":
			fillBar(stat.X, stat.Y, stat.Width-filledSize, stat.Height, bgClr)
			fillBar(stat.X+stat.Width-filledSize, stat.Y, filledSize, stat.Height, stat.BarColor)
		case "up":
			fillBar(stat.X, stat.Y, stat.Width, stat.Height-filledSize, bgClr)
			fillBar(stat.X, stat.Y+stat.Height-filledSize, stat.Width, filledSize, stat.BarColor)
		case "down":
			fillBar(stat.X, stat.Y+filledSize, stat.Width, stat.Height-filledSize, bgClr)
			fillBar(stat.X, stat.Y, stat.Width, filledSize, stat.BarColor)
		}
	}

	if stat.BorderWidth > 0 {
		strokeBorderNRGBA(frame, stat.X, stat.Y, stat.Width, stat.Height, stat.BorderWidth, stat.BarColor)
	} else if stat.BarOutline {
		strokeNRGBA(frame, stat.X, stat.Y, stat.Width, stat.Height, stat.BarColor)
	}
}

func (c *Compositor) drawSegmentedBarOnFrame(frame *image.NRGBA, stat *theme.Graph, direction string, filledSize, steps int, fillBar func(x, y, w, h int, clr color.Color), bgClr color.Color) {
	gap := stat.StepGap

	switch direction {
	case "left", "right":
		segW := stat.BlockWidth
		if segW <= 0 {
			segW = (stat.Width - (steps-1)*gap) / steps
		}
		if segW < 1 {
			segW = 1
		}
		filledSteps := int(math.Round(float64(filledSize) / float64(stat.Width) * float64(steps)))
		for i := 0; i < steps; i++ {
			var sx int
			if direction == "left" {
				sx = stat.X + i*(segW+gap)
			} else {
				sx = stat.X + stat.Width - (i+1)*(segW+gap) + gap
			}
			if i < filledSteps {
				fillBar(sx, stat.Y, segW, stat.Height, stat.BarColor)
			} else {
				fillBar(sx, stat.Y, segW, stat.Height, bgClr)
			}
		}
	case "up", "down":
		segH := stat.BlockWidth
		if segH <= 0 {
			segH = (stat.Height - (steps-1)*gap) / steps
		}
		if segH < 1 {
			segH = 1
		}
		filledSteps := int(math.Round(float64(filledSize) / float64(stat.Height) * float64(steps)))
		for i := 0; i < steps; i++ {
			var sy int
			if direction == "up" {
				sy = stat.Y + stat.Height - (i+1)*(segH+gap) + gap
			} else {
				sy = stat.Y + i*(segH+gap)
			}
			if i < filledSteps {
				fillBar(stat.X, sy, stat.Width, segH, stat.BarColor)
			} else {
				fillBar(stat.X, sy, stat.Width, segH, bgClr)
			}
		}
	}
}

func (c *Compositor) drawRadialOnFrame(frame *image.NRGBA, value float64, stat *theme.Radial) {
	x, y := float64(stat.X), float64(stat.Y)

	ratio := value / float64(stat.MaxValue-stat.MinValue)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}

	// Offset by -90° so AngleStart=0 → 12 o'clock (top).
	amin := utils.Radians(stat.AngleStart - 90)
	amax := utils.Radians(180 + stat.AngleStart - 90 + stat.AngleEnd)
	totalArc := amax - amin
	// Clamp full-circle configs (AngleEnd=360 yields totalArc=3π due to the
	// formula; cap at 2π so the arc never exceeds one revolution).
	if totalArc > 2*math.Pi-0.001 {
		totalArc = 2 * math.Pi
		amax = amin + totalArc
	}
	filledAngle := totalArc * ratio

	var cur float64
	if stat.RevertValue {
		cur = amax - filledAngle
	} else {
		cur = amin + filledAngle
	}
	if cur > amax {
		cur = amax
	}

	arcRadius := float64(stat.Radius - (stat.Width / 2))

	barClr := stat.BarColor
	var emptyClr color.Color
	if bgClr := stat.BackgroundStyle.BackgroundColor; bgClr != nil {
		if _, _, _, a := bgClr.RGBA(); a > 0 {
			emptyClr = bgClr
		}
	}
	if stat.Revert {
		barClr, emptyClr = emptyClr, barClr
	}

	hasGradient := stat.GradientColor != nil

	drawArc := func(a1, a2 float64, clr color.Color) {
		if clr == nil || a2 <= a1 {
			return
		}
		if hasGradient {
			drawArcGradientNRGBA(frame, x, y, arcRadius, a1, a2, stat.Width, stat.GradientColor, clr)
		} else {
			drawArcNRGBA(frame, x, y, arcRadius, a1, a2, stat.Width, clr)
		}
		if stat.Round {
			r := float64(stat.Width) / 2.0
			sx := x + arcRadius*math.Cos(a1)
			sy := y + arcRadius*math.Sin(a1)
			ex := x + arcRadius*math.Cos(a2)
			ey := y + arcRadius*math.Sin(a2)
			drawCircleNRGBA(frame, sx, sy, r, clr)
			drawCircleNRGBA(frame, ex, ey, r, clr)
		}
	}

	// Determine effective step count (BlockAngle takes priority over AngleSteps)
	angleSteps := stat.AngleSteps
	if stat.BlockAngle > 0 {
		blockAngleRad := utils.Radians(stat.BlockAngle)
		angleSteps = int(math.Round(totalArc / blockAngleRad))
		if angleSteps < 1 {
			angleSteps = 1
		}
	}

	if angleSteps > 1 && stat.AngleSep > 0 {
		segAngle := totalArc / float64(angleSteps)
		sepAngle := utils.Radians(stat.AngleSep)
		filledSteps := int(math.Round(ratio * float64(angleSteps)))
		if filledSteps > angleSteps {
			filledSteps = angleSteps
		}
		for i := 0; i < angleSteps; i++ {
			segStart := amin + float64(i)*segAngle
			segEnd := segStart + segAngle - sepAngle
			if segEnd <= segStart {
				continue
			}
			if stat.RevertValue {
				if i >= angleSteps-filledSteps {
					drawArc(segStart, segEnd, barClr)
				} else if emptyClr != nil {
					drawArc(segStart, segEnd, emptyClr)
				}
			} else {
				if i < filledSteps {
					drawArc(segStart, segEnd, barClr)
				} else if emptyClr != nil {
					drawArc(segStart, segEnd, emptyClr)
				}
			}
		}
	} else {
		if stat.RevertValue {
			if emptyClr != nil {
				drawArc(amin, cur, emptyClr)
			}
			drawArc(cur, amax, barClr)
		} else {
			if emptyClr != nil {
				drawArc(amin, amax, emptyClr)
			}
			drawArc(amin, cur, barClr)
		}
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

func toNRGBAColor(c color.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func lerpNRGBA(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(float64(a.R) + t*(float64(b.R)-float64(a.R))),
		G: uint8(float64(a.G) + t*(float64(b.G)-float64(a.G))),
		B: uint8(float64(a.B) + t*(float64(b.B)-float64(a.B))),
		A: uint8(float64(a.A) + t*(float64(b.A)-float64(a.A))),
	}
}

func fillGradientNRGBA(dst *image.NRGBA, x, y, w, h int, top, bottom color.Color) {
	nt := toNRGBAColor(top)
	nb := toNRGBAColor(bottom)
	bounds := dst.Bounds()
	for py := y; py < y+h; py++ {
		if py < bounds.Min.Y || py >= bounds.Max.Y {
			continue
		}
		t := 0.0
		if h > 1 {
			t = float64(py-y) / float64(h-1)
		}
		nc := lerpNRGBA(nt, nb, t)
		for px := x; px < x+w; px++ {
			if px < bounds.Min.X || px >= bounds.Max.X {
				continue
			}
			dst.SetNRGBA(px, py, nc)
		}
	}
}

func inRoundCorner(px, py, x, y, w, h, radius int) bool {
	r := float64(radius)
	dx := float64(px - x)
	dy := float64(py - y)
	rx := float64(x + w - 1 - px)
	ry := float64(y + h - 1 - py)
	if dx < r && dy < r {
		if (r-dx-0.5)*(r-dx-0.5)+(r-dy-0.5)*(r-dy-0.5) > r*r {
			return true
		}
	}
	if rx < r && dy < r {
		if (r-rx-0.5)*(r-rx-0.5)+(r-dy-0.5)*(r-dy-0.5) > r*r {
			return true
		}
	}
	if dx < r && ry < r {
		if (r-dx-0.5)*(r-dx-0.5)+(r-ry-0.5)*(r-ry-0.5) > r*r {
			return true
		}
	}
	if rx < r && ry < r {
		if (r-rx-0.5)*(r-rx-0.5)+(r-ry-0.5)*(r-ry-0.5) > r*r {
			return true
		}
	}
	return false
}

func fillRoundedNRGBA(dst *image.NRGBA, x, y, w, h, radius int, c color.Color) {
	nc := toNRGBAColor(c)
	bounds := dst.Bounds()
	for py := y; py < y+h; py++ {
		if py < bounds.Min.Y || py >= bounds.Max.Y {
			continue
		}
		for px := x; px < x+w; px++ {
			if px < bounds.Min.X || px >= bounds.Max.X {
				continue
			}
			if !inRoundCorner(px, py, x, y, w, h, radius) {
				dst.SetNRGBA(px, py, nc)
			}
		}
	}
}

func fillGradientRoundedNRGBA(dst *image.NRGBA, x, y, w, h, radius int, top, bottom color.Color) {
	nt := toNRGBAColor(top)
	nb := toNRGBAColor(bottom)
	bounds := dst.Bounds()
	for py := y; py < y+h; py++ {
		if py < bounds.Min.Y || py >= bounds.Max.Y {
			continue
		}
		t := 0.0
		if h > 1 {
			t = float64(py-y) / float64(h-1)
		}
		nc := lerpNRGBA(nt, nb, t)
		for px := x; px < x+w; px++ {
			if px < bounds.Min.X || px >= bounds.Max.X {
				continue
			}
			if !inRoundCorner(px, py, x, y, w, h, radius) {
				dst.SetNRGBA(px, py, nc)
			}
		}
	}
}

func strokeBorderNRGBA(dst *image.NRGBA, x, y, w, h, bw int, c color.Color) {
	for i := 0; i < bw; i++ {
		strokeNRGBA(dst, x+i, y+i, w-2*i, h-2*i, c)
	}
}

func drawArcGradientNRGBA(dst *image.NRGBA, cx, cy, radius, startAngle, endAngle float64, width int, top, bottom color.Color) {
	nt := toNRGBAColor(top)
	nb := toNRGBAColor(bottom)
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

	// See drawArcNRGBA: a full revolution collapses under the modulo, so draw
	// the whole ring when the span is ~2π.
	fullCircle := endAngle-startAngle >= 2*math.Pi-0.001
	totalH := float64(maxY - minY)
	twoPi := 2 * math.Pi
	for py := minY; py < maxY; py++ {
		for px := minX; px < maxX; px++ {
			dx := float64(px) + 0.5 - cx
			dy := float64(py) + 0.5 - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < innerR || dist > outerR {
				continue
			}
			if !fullCircle {
				angle := math.Mod(math.Atan2(dy, dx)+twoPi, twoPi)
				s := math.Mod(startAngle+twoPi, twoPi)
				e := math.Mod(endAngle+twoPi, twoPi)
				inArc := false
				if s <= e {
					if angle >= s && angle <= e {
						inArc = true
					}
				} else {
					if angle >= s || angle <= e {
						inArc = true
					}
				}
				if !inArc {
					continue
				}
			}
			t := 0.0
			if totalH > 0 {
				t = float64(py-minY) / totalH
			}
			dst.SetNRGBA(px, py, lerpNRGBA(nt, nb, t))
		}
	}
}

func fillNRGBA(dst *image.NRGBA, x, y, w, h int, c color.Color) {
	r, g, b, a := c.RGBA()
	nr, ng, nb, na := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
	bounds := dst.Bounds()

	// Clamp to bounds
	x1 := x
	if x1 < bounds.Min.X {
		x1 = bounds.Min.X
	}
	y1 := y
	if y1 < bounds.Min.Y {
		y1 = bounds.Min.Y
	}
	x2 := x + w
	if x2 > bounds.Max.X {
		x2 = bounds.Max.X
	}
	y2 := y + h
	if y2 > bounds.Max.Y {
		y2 = bounds.Max.Y
	}
	if x1 >= x2 || y1 >= y2 {
		return
	}

	actualW := x2 - x1

	// Fill first row directly into Pix
	firstOff := dst.PixOffset(x1, y1)
	for i := 0; i < actualW; i++ {
		off := firstOff + i*4
		dst.Pix[off] = nr
		dst.Pix[off+1] = ng
		dst.Pix[off+2] = nb
		dst.Pix[off+3] = na
	}

	// Copy first row to all remaining rows
	firstRow := dst.Pix[firstOff : firstOff+actualW*4]
	for py := y1 + 1; py < y2; py++ {
		rowOff := dst.PixOffset(x1, py)
		copy(dst.Pix[rowOff:rowOff+actualW*4], firstRow)
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

	// A full (or near-full) revolution collapses under the [0, 2π) modulo below:
	// start and end both map to the same angle, so the arc would render as a
	// single point. Detect it up front and draw the whole ring. This is what
	// makes a 0–360° radial's background actually appear.
	fullCircle := endAngle-startAngle >= 2*math.Pi-0.001
	twoPi := 2 * math.Pi
	for py := minY; py < maxY; py++ {
		for px := minX; px < maxX; px++ {
			dx := float64(px) + 0.5 - cx
			dy := float64(py) + 0.5 - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < innerR || dist > outerR {
				continue
			}
			if fullCircle {
				dst.SetNRGBA(px, py, nc)
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
