package local

import (
	"fmt"
	"git.sr.ht/~sbinet/gg"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
	"github.com/disintegration/gift"
	"image"
	"image/color"
	"os"
	"strings"
)

type Builder struct {
	log *logger.Logger
}

func NewBuilder(l *logger.Logger) *Builder {
	return &Builder{
		log: l,
	}
}

const tolerance = float64(5)

func (b *Builder) BuildBackgroundImage(images map[string]entity.StaticImage) image.Image {

	background, err := utils.LoadImage(images["background"].Path)
	if err != nil {
		b.log.Fatalf("error open file: %s", err)
		os.Exit(-1)
	}
	ctx := gg.NewContextForImage(background)

	for name, img := range images {
		if name == "background" {
			continue
		}
		numb, err := utils.LoadImage(img.Path)
		if err != nil {
			b.log.Fatalf("error open file %s: %s", name, err)
			os.Exit(-1)
		}
		b.log.Debugf("Build Background [%s] X:%d Y:%d Size (%dx%d)\n", name, img.X, img.Y, numb.Bounds().Dx(), numb.Bounds().Dy())

		ctx.DrawImage(numb, img.X, img.Y)
	}
	return ctx.Image()
}

func (b *Builder) BuildBackgroundTexts(background image.Image, images map[string]entity.StaticText) image.Image {
	ctx := gg.NewContextForImage(background)

	for _, text := range images {
		ctx.SetFontFace(text.Font)
		w, h := ctx.MeasureString(text.Text)

		x, y, x1, y1 := float64(text.X)-tolerance, float64(text.Y), w+tolerance, h

		if text.BackgroundColor != color.Transparent {
			ctx.SetColor(text.BackgroundColor)
			ctx.DrawRectangle(x, y, x1, y1)
			ctx.Fill()
		}
		b.log.Debugf("[%s] len:%d X:%d Y:%d Size (%.2f x %.2f)", text.Text, len(text.Text), text.X, text.Y, w, h)

		ctx.SetColor(text.FontColor)
		ctx.DrawStringAnchored(text.Text, float64(text.X)-(tolerance/2), float64(text.Y)-(tolerance/2), 0.0, 1.0)

		ctx.Fill()
	}
	return ctx.Image()
}

func (b *Builder) DrawText(background image.Image, text string, stat entity.StatText) image.Image {
	ctx := gg.NewContextForImage(background)

	ctx.SetFontFace(stat.Font)
	ctx.SetColor(stat.FontColor)
	ctx.ClearPath()

	measure := fmt.Sprintf("%s", strings.Repeat("0", stat.Padding))
	maxw, maxh := ctx.MeasureString(measure)

	w, h := ctx.MeasureString(text)

	center_total := (float64(stat.X) + maxw) / 2
	center_image := (float64(stat.X) + w) / 2
	center := center_total - center_image

	b.log.Debugf("FontHeight: %.2f", ctx.FontHeight())
	b.log.Debugf("[%s] len:%d X:%d Y:%d Size (%.2f x %.2f)", text, len(text), stat.X, stat.Y, w, h)

	if stat.Align == entity.CENTER {
		ctx.DrawStringAnchored(text, float64(stat.X)+center, float64(stat.Y), 0.0, 1.0)
	} else if stat.Align == entity.LEFT {
		ctx.DrawStringAnchored(text, float64(stat.X), float64(stat.Y), 0.0, 1.0)
	} else if stat.Align == entity.RIGHT {
		ctx.DrawStringAnchored(text, float64(stat.X)+maxw, float64(stat.Y), 1.0, 1.0)
	}
	ctx.Fill()
	ii := ctx.Image()

	crp := image.Rect(stat.X, stat.Y, stat.X+int(maxw), stat.Y+int(maxh))

	g := gift.New(
		gift.Crop(crp),
	)
	dst := image.NewRGBA(image.Rect(0, 0, int(maxw), int(maxh)))
	g.Draw(dst, ii)
	return dst
}

func (b *Builder) saveImage(img image.Image, file string) {
	ctx := gg.NewContextForImage(img)
	err := ctx.SavePNG(file)
	if err != nil {
		b.log.Infof("error saving file: %s\n", err)
	}
}

/*

func (b *Builder) DrawText(background image.Image, text entity.StaticText) image.Image {
	ctx := gg.NewContextForImage(background)

	ctx.SetFontFace(text.Font)
	ctx.SetColor(text.FontColor)
	//w, h := ctx.MeasureString(strings.Repeat("8", 4))
	str := fmt.Sprintf("%5s", text.Text)

	fmt.Printf("Font: %d\n", ctx.FontHeight())
	w, h := ctx.MeasureString(text.Text)
	w1, h1 := ctx.MeasureString(strings.Repeat("0", 4))

	b.log.Debugf("[%d] - [%s][%d x %d][%f x %f][%f x %f]\n", len(str), str, text.X, text.Y, w, h, w1, h1)

	//Alinhado a DIREITA
	//ctx.DrawStringAnchored(str, float64(text.X)+w1, float64(text.Y)+h1, 1.0, 0.0)

	//Alinhado ao CENTRO
	ctx.DrawStringAnchored(str, float64(text.X)+(w/2), float64(text.Y)+(h/2), 0.5, 0.5)

	ctx.Fill()
	ii := ctx.Image()

	crp := image.Rect(text.X, text.Y, text.X+int(w1), text.Y+int(h1))

	g := gift.New(
		gift.Crop(crp),
	)
	dst := image.NewRGBA(image.Rect(0, 0, int(w1), int(h1)))
	g.Draw(dst, ii)
	return dst
}


func BuildBackgroundImage(images map[string]entity.StaticImage) image.Image {

	background, err := utils.LoadImage(images["background"].Path)
	if err != nil {
		fmt.Printf("error open file: %s", err)
		os.Exit(-1)
	}
	dst := image.NewRGBA(image.Rect(images["background"].X, images["background"].Y, images["background"].Width, images["background"].Height))

	ctx := gg.NewContextForImage(background)

	for name, img := range images {
		if name == "background" {
			continue
		}
		numb, err := utils.LoadImage(img.Path)
		if err != nil {
			fmt.Printf("error open file: %s", err)
			os.Exit(-1)
		}
		ctx.DrawImage(numb, img.X, img.Y)
		ii := ctx.Image()
		crp := image.Rect(x, y, 140+x, 140+y)

		g := gift.New(
			gift.Crop(crp),
		)

		g.Draw(dst, ii)
	}
	return dst
}

*/
