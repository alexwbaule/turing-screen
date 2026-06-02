package sensors

import (
	"fmt"
	"time"

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
	img := builder.DrawText(str, text, defaultSize)
	return device.NewImageProcess(img), text.X, text.Y
}
