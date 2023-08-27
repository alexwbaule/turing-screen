package local

import (
	"fmt"
	"github.com/alexwbaule/gg"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
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
	log *logger.Logger
}

func NewBuilder(l *logger.Logger) *Builder {
	return &Builder{
		log: l,
	}
}

const tolerance = float64(2)
const border = float64(2)

func (b *Builder) BuildBackgroundImage(images map[string]theme.StaticImage) image.Image {
	ctx := gg.NewContextForImage(image.NewRGBA(image.Rect(0, 0, 800, 480)))

	keys := maps.Keys(images)
	slices.Sort(keys)
	for _, name := range keys {
		img := images[name]
		numb, err := utils.LoadImage(img.Path)
		if err != nil {
			b.log.Fatalf("error open file %s: %s", name, err)
			os.Exit(-1)
		}
		b.log.Debugf("Build Background Images [%s] X:%d Y:%d Size (%dx%d)", name, img.X, img.Y, numb.Bounds().Dx(), numb.Bounds().Dy())

		ctx.DrawImage(numb, img.X, img.Y)
	}
	return ctx.Image()
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
		b.log.Debugf("Build Background Texts [%s] len:%d X:%d Y:%d Size (%.2f x %.2f)", text.Text, len(text.Text), text.X, text.Y, w, h)

		ctx.SetColor(text.FontColor)
		ctx.DrawStringAnchored(text.Text, float64(text.X)-(tolerance/2), float64(text.Y)-(tolerance/2), 0.0, 1.0)

		ctx.Fill()
	}
	return ctx.Image()
}

func (b *Builder) DrawText(text string, stat *theme.Text) image.Image {
	ctx := gg.NewContextForImage(stat.BackgroundImage)

	ctx.SetFontFace(stat.Font)
	ctx.SetColor(stat.FontColor)
	ctx.ClearPath()

	measure := fmt.Sprintf("%s", strings.Repeat("8", utils.CountStr(text)))
	maxw, maxh := ctx.MeasureString(measure)

	w, h := ctx.MeasureString(text)

	center_total := (float64(stat.X) + maxw) / 2
	center_image := (float64(stat.X) + w) / 2
	center := center_total - center_image

	b.log.Debugf("Drawing Text [%s] len:%d Font:%.2f X:%d Y:%d Size (%.2f x %.2f) (%.2f x %.2f)", text, utils.CountStr(text), ctx.FontHeight(), stat.X, stat.Y, w, h, maxw, maxh)

	if stat.Align == theme.CENTER {
		ctx.DrawStringAnchored(text, float64(stat.X)+center, float64(stat.Y), 0.0, 1.0)
	} else if stat.Align == theme.LEFT {
		ctx.DrawStringAnchored(text, float64(stat.X)-1, float64(stat.Y), 0.0, 1.0)
	} else if stat.Align == theme.RIGHT {
		ctx.DrawStringAnchored(text, float64(stat.X)+maxw, float64(stat.Y), 1.0, 1.0)
	}
	ctx.Fill()
	ii := ctx.Image()

	x1, y1 := int(math.Round(maxw)), int(math.Round(maxh))

	b.log.Debugf("Drawing Text [%s] %dx%d", text, x1, y1)

	crp := image.Rect(stat.X, stat.Y, stat.X+x1, stat.Y+y1)

	g := gift.New(
		gift.Crop(crp),
	)
	dst := image.NewRGBA(image.Rect(0, 0, x1, y1))
	g.Draw(dst, ii)
	//b.saveImage(ii, fmt.Sprintf("res/test/image-ii-%s-%d-%d-%d-%.2fx%.2f-%.2fx%.2f.png", strings.Replace(strconv.Quote(text), "/", "-", -1), len(text), stat.X, stat.Y, w, h, maxw, maxh))
	//b.saveImage(dst, fmt.Sprintf("res/test/image-%s-%d-%d-%d-%.2fx%.2f-%.2fx%.2f.png", strings.Replace(strconv.Quote(text), "/", "-", -1), len(text), stat.X, stat.Y, w, h, maxw, maxh))
	return dst
}

func (b *Builder) DrawProgressBar(value float64, stat *theme.Graph) image.Image {
	ctx := gg.NewContextForImage(stat.BackgroundImage)

	barFilledWidth := math.Round(value / float64(stat.MaxValue-stat.MinValue) * float64(stat.Width))

	x, y, x1, y1 := float64(stat.X), float64(stat.Y), float64(stat.Width), float64(stat.Height)

	if stat.BarOutline {
		x, y, x1, y1 := float64(stat.X)-border, float64(stat.Y)-border, float64(stat.Width)+border, float64(stat.Height)+border
		b.log.Debugf("Drawing ProgressBar Size Outline (%.2f x %.2f) (%.2f x %.2f)", x, y, x1, y1)
		ctx.SetColor(stat.BarColor)
		ctx.DrawRectangle(x, y, x1, y1)
		ctx.Fill()
	}
	ctx.SetColor(stat.BarColor)
	ctx.DrawRectangle(x, y, barFilledWidth, y1)
	ctx.Fill()

	b.log.Debugf("Drawing ProgressBar Filled: %.2f  (%.2f x %.2f) (%.2f x %.2f)", barFilledWidth, x, y, x1, y1)

	ii := ctx.Image()

	crp := image.Rect(stat.X, stat.Y, stat.X+stat.Width, stat.Y+stat.Height)

	g := gift.New(
		gift.Crop(crp),
	)
	dst := image.NewRGBA(image.Rect(0, 0, stat.Width, stat.Height))
	g.Draw(dst, ii)
	//b.saveImage(dst, fmt.Sprintf("res/test/image-pb-%.0f-%dx%d-%dx%d.png", value, stat.X, stat.Y, stat.Width, stat.Height))
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
