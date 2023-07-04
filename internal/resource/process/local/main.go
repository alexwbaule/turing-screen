package local

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
	"github.com/disintegration/gift"
	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
	"image"
	"image/color"
	"os"
)

func BuildBackgroundImage(images map[string]entity.StaticImages) image.Image {

	background, err := utils.LoadImage(images["background"].Path)
	if err != nil {
		fmt.Printf("error open file: %s", err)
		os.Exit(-1)
	}
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
	}
	return ctx.Image()
}

func BuildBackgroundTexts(background image.Image, images map[string]entity.StaticTexts) image.Image {
	ctx := gg.NewContextForImage(background)

	for _, text := range images {
		err := ctx.LoadFontFace(text.Font, float64(text.FontSize))
		if err != nil {
			fmt.Printf("error loading font file: %s\n", err)
			//return nil
			ctx.SetFontFace(basicfont.Face7x13)
		}
		ctx.SetColor(color.White)

		ctx.DrawString(text.Text, float64(text.X), float64(text.Y))
		//im := DrawText(background, text)
		//fmt.Printf("Doing: %s\n", text.Text)
		//p := im.Bounds().Size().Y
		//ctx.DrawImage(im, text.X, text.Y-p)
	}
	return ctx.Image()
}

func DrawText(background image.Image, text entity.StaticTexts) image.Image {
	ctx := gg.NewContextForImage(background)

	err := ctx.LoadFontFace(text.Font, float64(text.FontSize))
	if err != nil {
		fmt.Printf("error loading font file: %s\n", err)
		//return nil
		ctx.SetFontFace(basicfont.Face7x13)
	}
	ctx.SetColor(color.White)
	//w, h := ctx.MeasureString(strings.Repeat("8", 4))
	w, h := ctx.MeasureString(text.Text)

	ctx.DrawStringAnchored(text.Text, float64(text.X)+(w/2), float64(text.Y)+(h/2), 0.5, 0.5)

	ii := ctx.Image()

	crp := image.Rect(text.X, text.Y, text.X+int(w)+2, text.Y+int(h)+2)

	g := gift.New(
		gift.Crop(crp),
	)

	dst := image.NewRGBA(image.Rect(0, 0, int(w)+2, int(h)+2))

	g.Draw(dst, ii)

	file := fmt.Sprintf("res/test/%s-%dx%d-%dx%d.png", text.Text, text.X, text.Y, text.X+int(w), text.Y-int(h))
	file2 := fmt.Sprintf("res/test/f-%s-%dx%d-%dx%d.png", text.Text, text.X, text.Y, text.X+int(w), text.Y-int(h))

	fmt.Printf("Saving: %s\n", file)
	ctx.SavePNG(file2)

	err = gg.SavePNG(file, dst)
	if err != nil {
		fmt.Printf("error saving file: %s\n", err)
		return nil
	}

	return dst
}

/*

func BuildBackgroundImage(images map[string]entity.StaticImages) image.Image {

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
