package local

import (
	"fmt"
	"github.com/alexwbaule/gg"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/disintegration/gift"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"image"
	"image/color"
	"math"
	"os"
	"strings"
)

type Builder struct {
	log    *logger.Logger
	device *device.Display
	theme  *theme.Display
}

func NewBuilder(l *logger.Logger, v *device.Display, d *theme.Display) *Builder {
	return &Builder{
		log:    l,
		device: v,
		theme:  d,
	}
}

const tolerance = float64(2)
const border = float64(2)

func (b *Builder) BuildBackgroundImage(images map[string]theme.StaticImage) image.Image {
	var numb image.Image

	if b.theme.Orientation == theme.PORTRAIT || b.theme.Orientation == theme.REVERSE_PORTRAIT {
		numb = image.NewRGBA(image.Rect(0, 0, b.device.Height, b.device.Width))
	} else {
		numb = image.NewRGBA(image.Rect(0, 0, b.device.Width, b.device.Height))
	}
	ctx := gg.NewContextForImage(numb)

	keys := maps.Keys(images)
	slices.Sort(keys)
	for _, name := range keys {
		img := images[name]
		numb, err := utils.LoadImage(img.Path)
		if err != nil {
			b.log.Fatalf("error open file %s: %s", name, err)
			os.Exit(-1)
		}
		//b.log.Debugf("Build Background Images [%s] X:%d Y:%d Size (%dx%d)", name, img.X, img.Y, numb.Bounds().Dx(), numb.Bounds().Dy())

		ctx.DrawImage(numb, img.X, img.Y)
	}
	img := ctx.Image()

	return img
}

func (b *Builder) BuildBackgroundTexts(background image.Image, images map[string]theme.StaticText) image.Image {
	ctx := gg.NewContextForImage(background)

	keys := maps.Keys(images)
	slices.Sort(keys)
	for _, name := range keys {
		text := images[name]
		ctx.SetFontFace(text.Font)
		w, h := ctx.MeasureString(text.Text)

		x, y, x1, y1 := float64(text.X)-tolerance, float64(text.Y), w+tolerance, h

		if text.BackgroundColor != color.Transparent {
			ctx.SetColor(text.BackgroundColor)
			ctx.DrawRectangle(x, y, x1, y1)
			ctx.Fill()
		}
		////b.log.Debugf("Build Background Texts [%s] len:%d X:%d Y:%d Size (%.2f x %.2f)", text.Text, len(text.Text), text.X, text.Y, w, h)

		ctx.SetColor(text.FontColor)
		ctx.DrawStringAnchored(text.Text, float64(text.X)-(tolerance/2), float64(text.Y)-(tolerance/2), 0.0, 1.0)
	}
	numb := ctx.Image()

	////b.saveImage(numb, fmt.Sprintf("res/test/image-texts.png"))
	return numb
}

func (b *Builder) DrawText(text string, stat *theme.Text) image.Image {
	var numb image.Image

	if stat.BackgroundImage == nil {
		if b.theme.Orientation == theme.PORTRAIT || b.theme.Orientation == theme.REVERSE_PORTRAIT {
			numb = utils.CreateImage(b.device.Height, b.device.Width, stat.BackgroundColor)
		} else {
			numb = utils.CreateImage(b.device.Width, b.device.Height, stat.BackgroundColor)
		}
	} else {
		numb = stat.BackgroundImage
	}

	ctx := gg.NewContextForImage(numb)

	ctx.SetFontFace(stat.Font)
	ctx.SetColor(stat.FontColor)
	ctx.ClearPath()

	measure := fmt.Sprintf("%s", strings.Repeat("8", utils.CountStr(text)))
	maxw, maxh := ctx.MeasureString(measure)

	w, _ := ctx.MeasureString(text)
	x1, y1 := int(math.Round(maxw)), int(math.Round(maxh))

	center_total := (float64(stat.X) + maxw) / 2
	center_image := (float64(stat.X) + w) / 2
	center := center_total - center_image

	//b.log.Debugf("Drawing Text [%s] len:%d Font:%.2f X:%d Y:%d Size (%.2f x %.2f) (%.2f x %.2f)", text, utils.CountStr(text), ctx.FontHeight(), stat.X, stat.Y, w, h, maxw, maxh)

	if stat.Align == theme.CENTER {
		ctx.DrawStringAnchored(text, float64(stat.X)+center, float64(stat.Y), 0.0, 1.0)
	} else if stat.Align == theme.LEFT {
		ctx.DrawStringAnchored(text, float64(stat.X)-1, float64(stat.Y), 0.0, 1.0)
	} else if stat.Align == theme.RIGHT {
		ctx.DrawStringAnchored(text, float64(stat.X)+maxw, float64(stat.Y), 1.0, 1.0)
	}

	ii := ctx.Image()
	//b.log.Debugf("Drawing Text [%s] %dx%d", text, x1, y1)

	crp := image.Rect(stat.X, stat.Y, stat.X+x1, stat.Y+y1)

	g := gift.New(
		gift.Crop(crp),
	)
	dst := image.NewRGBA(image.Rect(0, 0, x1, y1))
	g.Draw(dst, ii)

	//b.saveImage(ii, fmt.Sprintf("res/test/image-twisted-%s-%d-%d-%d-%.2fx%.2f-%.2fx%.2f.png", strings.Replace(strconv.Quote(text), "/", "-", -1), len(text), stat.X, stat.Y, w, h, maxw, maxh))
	//b.saveImage(dst, fmt.Sprintf("res/test/image-%s-%d-%d-%d-%.2fx%.2f-%.2fx%.2f.png", strings.Replace(strconv.Quote(text), "/", "-", -1), len(text), stat.X, stat.Y, w, h, maxw, maxh))
	return dst
}

func (b *Builder) DrawProgressBar(value float64, stat *theme.Graph) image.Image {
	var numb image.Image

	if stat.BackgroundImage == nil {
		if b.theme.Orientation == theme.PORTRAIT || b.theme.Orientation == theme.REVERSE_PORTRAIT {
			numb = utils.CreateImage(b.device.Height, b.device.Width, color.Transparent)
		} else {
			numb = utils.CreateImage(b.device.Width, b.device.Height, color.Transparent)
		}
	} else {
		numb = stat.BackgroundImage
	}

	ctx := gg.NewContextForImage(numb)
	barFilledWidth := math.Round(value / float64(stat.MaxValue-stat.MinValue) * float64(stat.Width))

	x, y, x1, y1 := float64(stat.X), float64(stat.Y), float64(stat.Width), float64(stat.Height)

	ctx.SetColor(stat.BarColor)
	ctx.DrawRectangle(x, y, barFilledWidth, y1)
	ctx.Fill()
	if stat.BarOutline {
		//b.log.Debugf("Drawing ProgressBar Size Outline (%.2f x %.2f) (%.2f x %.2f)", x, y, x1, y1)
		ctx.SetColor(stat.BarColor)
		//ctx.SetLineWidth(1)
		ctx.DrawRectangle(x, y, x1, y1)
		ctx.Stroke()
	}
	//b.log.Debugf("Drawing ProgressBar Filled: %.2f  (%.2f x %.2f) (%.2f x %.2f)", barFilledWidth, x, y, x1, y1)

	ii := ctx.Image()

	crp := image.Rect(stat.X, stat.Y, stat.X+stat.Width, stat.Y+stat.Height)

	g := gift.New(
		gift.Crop(crp),
	)
	dst := image.NewRGBA(image.Rect(0, 0, stat.Width, stat.Height))
	g.Draw(dst, ii)
	//b.saveImage(ii, fmt.Sprintf("res/test/image-pb-full-%.0f-%dx%d-%dx%d.png", value, stat.X, stat.Y, stat.Width, stat.Height))
	//b.saveImage(dst, fmt.Sprintf("res/test/image-pb-%.0f-%dx%d-%dx%d.png", value, stat.X, stat.Y, stat.Width, stat.Height))
	return dst
}

func (b *Builder) DrawRadialProgressBar(value float64, stat *theme.Radial) image.Image {
	var numb image.Image

	//stat.BarColor
	//stat.BackgroundColor
	//stat.FontColor

	if stat.BackgroundImage == nil {
		if b.theme.Orientation == theme.PORTRAIT || b.theme.Orientation == theme.REVERSE_PORTRAIT {
			numb = utils.CreateImage(b.device.Height, b.device.Width, stat.BackgroundColor)
		} else {
			numb = utils.CreateImage(b.device.Width, b.device.Height, stat.BackgroundColor)
		}
	} else {
		numb = stat.BackgroundImage
	}
	ctx := gg.NewContextForImage(numb)
	ctx.SetColor(stat.BarColor)

	if math.Mod(float64(stat.AngleStart), 631) == math.Mod(float64(stat.AngleEnd), 361) {
		if stat.Clockwise {
			stat.AngleStart += 1
		} else {
			stat.AngleEnd += 1
		}
	}

	diameter := 2 * stat.Radius
	x, y, _, _ := float64(stat.X), float64(stat.Y), float64(stat.X-stat.Radius), float64(stat.Y+stat.Radius)
	//percent := (value - float64(stat.MinValue)) / float64(stat.MaxValue-stat.MinValue)
	/*
		const S = 128

		for i := 0; i < 360; i += 15 {
			ctx.Push()
			ctx.RotateAbout(gg.Radians(float64(i)), S/2, S/2)
			ctx.DrawEllipse(S/2, S/2, S*7/16, S/8)
			ctx.Fill()
			ctx.Pop()
		}
	*/

	pct := (value - float64(stat.MinValue)) / float64(stat.MaxValue-stat.MinValue)
	crazy := stat.AngleEnd - stat.AngleStart

	v := float64(crazy) * pct

	b.log.Infof("Percent: %f [%f]", pct, v)

	a := utils.Radians(stat.AngleStart)
	c := utils.Radians(stat.AngleEnd)

	fmt.Printf("A: %f, C:%f\n", a, c)

	ctx.DrawCircle(float64(stat.X-stat.Radius), float64(stat.Y-stat.Radius), 5)
	ctx.DrawCircle(float64(stat.X+stat.Radius), float64(stat.Y+stat.Radius), 5)
	ctx.DrawCircle(float64(stat.X+stat.Radius), float64(stat.Y-stat.Radius), 5)
	ctx.DrawCircle(float64(stat.X-stat.Radius), float64(stat.Y+stat.Radius), 5)
	ctx.Fill()

	ctx.ClearPath()

	ctx.SetColor(stat.BarColor)

	//ctx.SetDash(10)
	//ctx.SetDashOffset(1)

	ctx.SetLineCapSquare()

	ctx.DrawArc(x, y, float64(stat.Radius-(stat.Width/2)), a, c)
	ctx.Rotate(90)

	ctx.NewSubPath()
	ctx.ClosePath()

	//ctx.DrawCircle(x, y, float64(stat.Radius))

	ctx.SetLineWidth(float64(stat.Width))
	ctx.Stroke()

	//ctx.Fill()

	ii := ctx.Image()

	crp := image.Rect(stat.X-stat.Radius, stat.Y-stat.Radius, stat.X+stat.Radius, stat.Y+stat.Radius)

	g := gift.New(
		gift.Crop(crp),
	)
	dst := image.NewRGBA(image.Rect(0, 0, diameter, diameter))
	g.Draw(dst, ii)
	//b.saveImage(ii, fmt.Sprintf("res/test/image-radial-full-%.0f-%dx%d-%dx%d.png", value, stat.X, stat.Y, stat.Width, stat.Radius))
	//zb.saveImage(dst, fmt.Sprintf("res/test/image-radial-%.0f-%dx%d-%dx%d.png", value, stat.X, stat.Y, stat.Width, stat.Radius))
	return dst
}

func (b *Builder) saveImage(img image.Image, file string) {
	ctx := gg.NewContextForImage(img)
	err := ctx.SavePNG(file)
	if err != nil {
		b.log.Infof("error saving file: %s\n", err)
	}
}
