package local

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strings"

	"github.com/alexwbaule/gg"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/disintegration/gift"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
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

// BuildTransparentBackground creates a fully transparent RGBA image.
// Used for video overlay mode — the video shows through transparent areas,
// and sensor data renders on top.
func (b *Builder) BuildTransparentBackground() image.Image {
	if b.theme.Orientation == theme.PORTRAIT || b.theme.Orientation == theme.REVERSE_PORTRAIT {
		return image.NewRGBA(image.Rect(0, 0, b.device.Height, b.device.Width))
	}
	return image.NewRGBA(image.Rect(0, 0, b.device.Width, b.device.Height))
}

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
	// Measure text dimensions using a temporary context
	tmpCtx := gg.NewContext(1, 1)
	tmpCtx.SetFontFace(stat.Font)

	measure := fmt.Sprintf("%s", strings.Repeat("8", utils.CountStr(text)))
	maxw, maxh := tmpCtx.MeasureString(measure)

	x1, y1 := int(math.Round(maxw)), int(math.Round(maxh))

	// Allocate image buffer at target dimensions (maxw x maxh)
	var numb image.Image

	if stat.BackgroundImage == nil {
		numb = utils.CreateImage(x1, y1, stat.BackgroundColor)
	} else {
		// Crop BackgroundImage to target region size at (stat.X, stat.Y)
		crpRect := image.Rect(stat.X, stat.Y, stat.X+x1, stat.Y+y1)
		cropped := image.NewRGBA(image.Rect(0, 0, x1, y1))
		g := gift.New(gift.Crop(crpRect))
		g.Draw(cropped, stat.BackgroundImage)
		numb = cropped
	}

	ctx := gg.NewContextForImage(numb)

	ctx.SetFontFace(stat.Font)
	ctx.SetColor(stat.FontColor)
	ctx.ClearPath()

	w, _ := ctx.MeasureString(text)

	centerTotal := maxw / 2
	centerImage := w / 2
	center := centerTotal - centerImage

	//b.log.Debugf("Drawing Text [%s] len:%d Font:%.2f X:%d Y:%d Size (%.2f x %.2f) (%.2f x %.2f)", text, utils.CountStr(text), ctx.FontHeight(), stat.X, stat.Y, w, h, maxw, maxh)

	// Draw text at (0, 0) within the allocated buffer
	if stat.Align == theme.CENTER {
		ctx.DrawStringAnchored(text, center, 0, 0.0, 1.0)
	} else if stat.Align == theme.LEFT {
		ctx.DrawStringAnchored(text, 0, 0, 0.0, 1.0)
	} else if stat.Align == theme.RIGHT {
		ctx.DrawStringAnchored(text, maxw, 0, 1.0, 1.0)
	}

	//b.log.Debugf("Drawing Text [%s] %dx%d", text, x1, y1)

	return ctx.Image()
}

func (b *Builder) DrawProgressBar(value float64, stat *theme.Graph) image.Image {
	var numb image.Image

	if stat.BackgroundImage == nil {
		numb = utils.CreateImage(stat.Width, stat.Height, color.Transparent)
	} else {
		// Crop BackgroundImage to target region size at (stat.X, stat.Y)
		cropped := utils.CreateImage(stat.Width, stat.Height, color.Transparent)
		cropCtx := gg.NewContextForImage(cropped)
		cropCtx.DrawImage(stat.BackgroundImage, -stat.X, -stat.Y)
		numb = cropCtx.Image()
	}

	ctx := gg.NewContextForImage(numb)
	barFilledWidth := math.Round(value / float64(stat.MaxValue-stat.MinValue) * float64(stat.Width))

	x, y, x1, y1 := float64(0), float64(0), float64(stat.Width), float64(stat.Height)

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

	dst := ctx.Image()
	//b.saveImage(dst, fmt.Sprintf("res/test/image-pb-%.0f-%dx%d-%dx%d.png", value, stat.X, stat.Y, stat.Width, stat.Height))
	return dst
}

func (b *Builder) DrawRadialProgressBar(value float64, stat *theme.Radial) image.Image {
	var numb image.Image

	diameter := 2 * stat.Radius

	if stat.BackgroundImage == nil {
		numb = utils.CreateImage(diameter, diameter, stat.BackgroundColor)
	} else {
		// Crop the background image to the target region size
		crp := image.Rect(stat.X-stat.Radius, stat.Y-stat.Radius, stat.X+stat.Radius, stat.Y+stat.Radius)
		cropped := image.NewRGBA(image.Rect(0, 0, diameter, diameter))
		g := gift.New(gift.Crop(crp))
		g.Draw(cropped, stat.BackgroundImage)
		numb = cropped
	}
	ctx := gg.NewContextForImage(numb)

	// Draw centered at (radius, radius) within the small buffer
	x, y := float64(stat.Radius), float64(stat.Radius)

	amin := utils.Radians(stat.AngleStart)
	amax := utils.Radians(180 + stat.AngleStart + stat.AngleEnd)

	total := (value * float64(180+stat.AngleStart+stat.AngleEnd)) / 100

	cur := utils.Radians(int(total) + stat.AngleStart)

	if cur > amax {
		cur = amax
	}

	b.log.Debugf("A: %f, C: %f %f", amin, amax, cur)

	if stat.ShowText {
		ctx.SetFontFace(stat.Font)
		ctx.SetColor(stat.FontColor)
		ctx.ClearPath()

		measure := fmt.Sprintf("%3.f", value)
		if stat.ShowUnit {
			measure = fmt.Sprintf("%3.f%%", value)
		}

		ctx.DrawStringAnchored(measure, float64(stat.Radius), float64(stat.Radius), 0.5, 0.5)
	}
	ctx.SetColor(stat.BarColor)

	ctx.SetLineCapSquare()

	ctx.DrawArc(x, y, float64(stat.Radius-(stat.Width/2)), amin, cur)
	ctx.NewSubPath()
	ctx.ClosePath()

	ctx.SetLineWidth(float64(stat.Width))
	ctx.Stroke()

	return ctx.Image()
}

func (b *Builder) saveImage(img image.Image, file string) {
	ctx := gg.NewContextForImage(img)
	err := ctx.SavePNG(file)
	if err != nil {
		b.log.Infof("error saving file: %s\n", err)
	}
}
