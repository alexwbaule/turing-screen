package sensors

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"time"
)

func BuildRadial(builder *renderer.Builder, mesurement float64, radial *theme.Radial) (*device.ImageProcess, int, int) {
	img := builder.DrawRadialProgressBar(mesurement, radial)
	return device.NewImageProcess(img), radial.X - radial.Radius, radial.Y - radial.Radius
}
func BuildGraph(builder *renderer.Builder, mesurement float64, graph *theme.Graph) (*device.ImageProcess, int, int) {
	img := builder.DrawProgressBar(mesurement, graph)
	return device.NewImageProcess(img), graph.X, graph.Y
}

func BuildText(builder *renderer.Builder, mesurement any, format string, unit string, text *theme.Text) (*device.ImageProcess, int, int) {
	str := fmt.Sprintf(format, mesurement)
	if text.ShowUnit {
		str += unit
	}
	return buildText(builder, str, text)
}

func BuildTextFloat(builder *renderer.Builder, mesurement float64, fn func(f float64, b bool) string, text *theme.Text) (*device.ImageProcess, int, int) {
	str := fn(mesurement, text.ShowUnit)
	return buildText(builder, str, text)
}

func BuildTextUint(builder *renderer.Builder, mesurement uint64, fn func(f uint64, b bool) string, text *theme.Text) (*device.ImageProcess, int, int) {
	str := fn(mesurement, text.ShowUnit)
	return buildText(builder, str, text)
}

func BuildTextDt(builder *renderer.Builder, mesurement time.Time, format theme.FormatDateTime, text *theme.Text) (*device.ImageProcess, int, int) {
	str := mesurement.Format(text.Format.String(format))
	return buildText(builder, str, text)
}

func buildText(builder *renderer.Builder, str string, text *theme.Text) (*device.ImageProcess, int, int) {
	img := builder.DrawText(str, text)
	return device.NewImageProcess(img), text.X, text.Y
}
