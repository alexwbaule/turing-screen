package ui

import (
	"strings"

	"github.com/alexwbaule/turing-screen/internal/theme"
)

// Available sensors for the Sensor select
var sensorOptions = []string{
	"CPU.PERCENTAGE",
	"CPU.TEMPERATURE",
	"CPU.FREQUENCY",
	"CPU.FAN",
	"CPU.POWER",
	"CPU.VOLTAGE",
	"GPU.PERCENTAGE",
	"GPU.TEMPERATURE",
	"GPU.MEMORY",
	"GPU.POWER",
	"GPU.FREQUENCY",
	"GPU.VOLTAGE",
	"GPU.FAN",
	"MEMORY.VIRTUAL",
	"MEMORY.SWAP",
	"DISK.USED",
	"DISK.FREE",
	"DISK.TOTAL",
	"DISK.TEMPERATURE",
	"NET.ETH.UPLOAD",
	"NET.ETH.DOWNLOAD",
	"NET.WLO.UPLOAD",
	"NET.WLO.DOWNLOAD",
	"DATE.HOUR",
	"DATE.DAY",
	"WEATHER.TEMPERATURE",
	"VOLUME",
}

// Available widget types for the Type select
var widgetTypeOptions = []string{
	"TEXT",
	"GRAPH",
	"RADIAL",
	"CHART",
	"STATUS_BAR",
	"GAUGE",
}

// parseSensorFromPath extracts the sensor path from a YAML path.
// Example: "STATS.CPU.PERCENTAGE.RADIAL" -> "CPU.PERCENTAGE"
func parseSensorFromPath(yamlPath string) string {
	// Remove "STATS." prefix
	path := strings.TrimPrefix(yamlPath, "STATS.")
	// Remove widget type suffix (.TEXT, .GRAPH, .RADIAL, .CHART, .STATUS_BAR, .GAUGE, .PERCENT_TEXT)
	for _, suffix := range []string{".TEXT", ".GRAPH", ".RADIAL", ".CHART", ".STATUS_BAR", ".GAUGE", ".PERCENT_TEXT"} {
		if strings.HasSuffix(path, suffix) {
			path = strings.TrimSuffix(path, suffix)
			break
		}
	}
	return path
}

// parseWidgetTypeFromPath extracts the widget type from a YAML path.
// Example: "STATS.CPU.PERCENTAGE.RADIAL" -> "RADIAL"
func parseWidgetTypeFromPath(yamlPath string) string {
	parts := strings.Split(yamlPath, ".")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	switch last {
	case "TEXT", "GRAPH", "RADIAL", "CHART", "STATUS_BAR", "GAUGE", "PERCENT_TEXT":
		return last
	}
	return ""
}

// getMeasurementForSensor returns a pointer to the Measurement struct for a sensor path.
// Creates intermediate structs if they don't exist.
func getMeasurementForSensor(t *theme.Theme, sensor string) *theme.Measurement {
	if t.Stats == nil {
		t.Stats = &theme.Stats{}
	}

	parts := strings.Split(sensor, ".")

	switch {
	case len(parts) >= 2 && parts[0] == "CPU":
		if t.Stats.CPU == nil {
			t.Stats.CPU = &theme.CPU{}
		}
		switch parts[1] {
		case "PERCENTAGE":
			if t.Stats.CPU.Percentage == nil {
				t.Stats.CPU.Percentage = &theme.Measurement{}
			}
			return t.Stats.CPU.Percentage
		case "TEMPERATURE":
			if t.Stats.CPU.Temperature == nil {
				t.Stats.CPU.Temperature = &theme.Measurement{}
			}
			return t.Stats.CPU.Temperature
		case "FREQUENCY":
			if t.Stats.CPU.Frequency == nil {
				t.Stats.CPU.Frequency = &theme.Measurement{}
			}
			return t.Stats.CPU.Frequency
		case "FAN":
			if t.Stats.CPU.Fan == nil {
				t.Stats.CPU.Fan = &theme.Measurement{}
			}
			return t.Stats.CPU.Fan
		case "POWER":
			if t.Stats.CPU.Power == nil {
				t.Stats.CPU.Power = &theme.Measurement{}
			}
			return t.Stats.CPU.Power
		case "VOLTAGE":
			if t.Stats.CPU.Voltage == nil {
				t.Stats.CPU.Voltage = &theme.Measurement{}
			}
			return t.Stats.CPU.Voltage
		}

	case len(parts) >= 2 && parts[0] == "GPU":
		if t.Stats.GPU == nil {
			t.Stats.GPU = &theme.GPU{}
		}
		switch parts[1] {
		case "PERCENTAGE":
			if t.Stats.GPU.Percentage == nil {
				t.Stats.GPU.Percentage = &theme.Measurement{}
			}
			return t.Stats.GPU.Percentage
		case "TEMPERATURE":
			if t.Stats.GPU.Temperature == nil {
				t.Stats.GPU.Temperature = &theme.Measurement{}
			}
			return t.Stats.GPU.Temperature
		case "MEMORY":
			if t.Stats.GPU.Memory == nil {
				t.Stats.GPU.Memory = &theme.Measurement{}
			}
			return t.Stats.GPU.Memory
		case "POWER":
			if t.Stats.GPU.Power == nil {
				t.Stats.GPU.Power = &theme.Measurement{}
			}
			return t.Stats.GPU.Power
		case "FREQUENCY":
			if t.Stats.GPU.Frequency == nil {
				t.Stats.GPU.Frequency = &theme.Measurement{}
			}
			return t.Stats.GPU.Frequency
		case "VOLTAGE":
			if t.Stats.GPU.Voltage == nil {
				t.Stats.GPU.Voltage = &theme.Measurement{}
			}
			return t.Stats.GPU.Voltage
		case "FAN":
			if t.Stats.GPU.Fan == nil {
				t.Stats.GPU.Fan = &theme.Measurement{}
			}
			return t.Stats.GPU.Fan
		}

	case len(parts) >= 2 && parts[0] == "DISK":
		if t.Stats.Disk == nil {
			t.Stats.Disk = &theme.Disk{}
		}
		switch parts[1] {
		case "USED":
			if t.Stats.Disk.Used == nil {
				t.Stats.Disk.Used = &theme.Measurement{}
			}
			return t.Stats.Disk.Used
		case "FREE":
			if t.Stats.Disk.Free == nil {
				t.Stats.Disk.Free = &theme.Measurement{}
			}
			return t.Stats.Disk.Free
		case "TOTAL":
			if t.Stats.Disk.Total == nil {
				t.Stats.Disk.Total = &theme.Measurement{}
			}
			return t.Stats.Disk.Total
		case "TEMPERATURE":
			if t.Stats.Disk.Temperature == nil {
				t.Stats.Disk.Temperature = &theme.Measurement{}
			}
			return t.Stats.Disk.Temperature
		}

	case len(parts) >= 2 && parts[0] == "DATE":
		if t.Stats.Date == nil {
			t.Stats.Date = &theme.DateTime{}
		}
		switch parts[1] {
		case "HOUR":
			if t.Stats.Date.Hour == nil {
				t.Stats.Date.Hour = &theme.Measurement{}
			}
			return t.Stats.Date.Hour
		case "DAY":
			if t.Stats.Date.Day == nil {
				t.Stats.Date.Day = &theme.Measurement{}
			}
			return t.Stats.Date.Day
		}

	case len(parts) >= 2 && parts[0] == "WEATHER":
		if t.Stats.Weather == nil {
			t.Stats.Weather = &theme.Weather{}
		}
		if parts[1] == "TEMPERATURE" {
			if t.Stats.Weather.Temperature == nil {
				t.Stats.Weather.Temperature = &theme.Measurement{}
			}
			return t.Stats.Weather.Temperature
		}

	case parts[0] == "VOLUME":
		// Volume uses Text directly, not Measurement — handle differently
		return nil
	}

	// Network needs special handling (3 levels)
	if len(parts) >= 3 && parts[0] == "NET" {
		if t.Stats.Net == nil {
			t.Stats.Net = &theme.Network{}
		}
		var nm *theme.NetworkMeasurement
		switch parts[1] {
		case "ETH":
			if t.Stats.Net.Wired == nil {
				t.Stats.Net.Wired = &theme.NetworkMeasurement{}
			}
			nm = t.Stats.Net.Wired
		case "WLO":
			if t.Stats.Net.Wifi == nil {
				t.Stats.Net.Wifi = &theme.NetworkMeasurement{}
			}
			nm = t.Stats.Net.Wifi
		}
		if nm != nil {
			switch parts[2] {
			case "UPLOAD":
				if nm.Upload == nil {
					nm.Upload = &theme.Measurement{}
				}
				return nm.Upload
			case "DOWNLOAD":
				if nm.Download == nil {
					nm.Download = &theme.Measurement{}
				}
				return nm.Download
			}
		}
	}

	return nil
}

// widgetSnapshot captures compatible fields when switching widget type or sensor.
type widgetSnapshot struct {
	X, Y            int
	Index           int
	MinValue        int
	MaxValue        int
	Width           int
	Height          int
	Radius          int
	ArcWidth        int
	AngleStart      int
	AngleEnd        int
	AngleSteps      int
	AngleSep        int
	BlockAngle      int
	Clockwise       bool
	Round           bool
	PrimaryColor    string
	BackgroundColor string
	GradientColor   string
	Font            string
	FontSize        int
	FontColor       string
}

// setWidgetOnMeasurement sets the appropriate widget type on a Measurement,
// clearing all other widget types, and applying fields from snap.
func setWidgetOnMeasurement(m *theme.Measurement, widgetType string, snap widgetSnapshot) {
	if snap.Font == "" {
		snap.Font = "jetbrains-mono/JetBrainsMono-Bold.ttf"
	}
	if snap.FontSize == 0 {
		snap.FontSize = 22
	}
	if snap.PrimaryColor == "" {
		snap.PrimaryColor = "255, 255, 255"
	}
	if snap.Width == 0 {
		snap.Width = 150
	}
	if snap.Height == 0 {
		snap.Height = 15
	}
	if snap.Radius == 0 {
		snap.Radius = 40
	}
	if snap.ArcWidth == 0 {
		snap.ArcWidth = 10
	}
	if snap.AngleEnd == 0 {
		snap.AngleStart = 36
		snap.AngleEnd = 60
	}
	if snap.AngleSteps == 0 {
		snap.AngleSteps = 1
	}
	if snap.MaxValue == 0 {
		snap.MaxValue = 100
	}

	m.Graph = nil
	m.Radial = nil
	m.Chart = nil
	m.StatusBar = nil
	m.Gauge = nil
	m.Text = nil

	switch widgetType {
	case "TEXT":
		m.Text = &theme.Text{
			Show: true, X: snap.X, Y: snap.Y, Index: snap.Index,
			Font: snap.Font, FontSize: snap.FontSize,
			FontColor:       snap.PrimaryColor,
			BackgroundColor: snap.BackgroundColor,
		}
	case "GRAPH":
		m.Graph = &theme.Graph{
			Show: true, X: snap.X, Y: snap.Y, Index: snap.Index,
			Width: snap.Width, Height: snap.Height,
			MinValue: snap.MinValue, MaxValue: snap.MaxValue,
			BarColor:        snap.PrimaryColor,
			BackgroundColor: snap.BackgroundColor,
			GradientColor:   snap.GradientColor,
		}
	case "RADIAL":
		m.Radial = &theme.Radial{
			Show: true, X: snap.X, Y: snap.Y, Index: snap.Index,
			Radius: snap.Radius, Width: snap.ArcWidth,
			MinValue: snap.MinValue, MaxValue: snap.MaxValue,
			AngleStart: snap.AngleStart, AngleEnd: snap.AngleEnd,
			AngleSteps: snap.AngleSteps, AngleSep: snap.AngleSep,
			BlockAngle: snap.BlockAngle,
			Clockwise:       snap.Clockwise,
			Round:           snap.Round,
			BarColor:        snap.PrimaryColor,
			BackgroundColor: snap.BackgroundColor,
			GradientColor:   snap.GradientColor,
			Font:            snap.Font,
			FontColor:       snap.FontColor,
		}
	case "CHART":
		m.Chart = &theme.Chart{
			Show: true, X: snap.X, Y: snap.Y, Index: snap.Index,
			Width: snap.Width, Height: snap.Height,
			Style: "line", MinValue: snap.MinValue, MaxValue: snap.MaxValue,
			ColumnWidth: 3, ColumnGap: 1,
			FillColor:   snap.PrimaryColor,
			LineColor:   "255, 255, 255",
			BorderWidth: 1,
		}
	case "STATUS_BAR":
		m.StatusBar = &theme.StatusBar{
			Show: true, X: snap.X, Y: snap.Y, Index: snap.Index,
			Width: snap.Width, Height: snap.Height,
			MinValue: snap.MinValue, MaxValue: snap.MaxValue,
			BarColor:        snap.PrimaryColor,
			BackgroundColor: snap.BackgroundColor,
			IndicatorColor:  "255, 255, 255",
			IndicatorRadius: 6,
		}
	case "GAUGE":
		m.Gauge = &theme.Gauge{
			Show: true, X: snap.X, Y: snap.Y, Index: snap.Index,
			Radius: snap.Radius, NeedleWidth: 2,
			MinValue: snap.MinValue, MaxValue: snap.MaxValue,
			AngleStart:      snap.AngleStart,
			AngleEnd:        snap.AngleEnd,
			NeedleColor:     snap.PrimaryColor,
			BackgroundColor: snap.BackgroundColor,
			Font:            snap.Font,
			FontColor:       snap.FontColor,
			ShowText:        true,
		}
	}
}
