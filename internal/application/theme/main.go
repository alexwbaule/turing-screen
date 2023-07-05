package theme

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/config"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
	"golang.org/x/image/font"
	"image/color"
)

type Theme struct {
	theme map[string]interface{}
	path  string
}

const fontPath = "res/fonts/"

func LoadTheme(themeFile string) (*Theme, error) {
	tfile := fmt.Sprintf("res/themes/%s/theme.yaml", themeFile)

	cfg, err := config.NewConfig(tfile)
	if err != nil {
		return nil, err
	}
	return &Theme{
		theme: cfg.AllSettings(),
		path:  fmt.Sprintf("res/themes/%s/", themeFile),
	}, nil
}

func (t Theme) GetStaticImages() map[string]entity.StaticImages {
	images := make(map[string]entity.StaticImages)

	if t.theme["static_images"] == nil {
		return images
	}

	for b, i := range t.theme["static_images"].(map[string]interface{}) {
		fmt.Printf("%s [%#v]\n", b, i)
		images[b] = entity.StaticImages{
			Height: i.(map[string]interface{})["height"].(int),
			Path:   t.path + i.(map[string]interface{})["path"].(string),
			Width:  i.(map[string]interface{})["width"].(int),
			X:      i.(map[string]interface{})["x"].(int),
			Y:      i.(map[string]interface{})["y"].(int),
		}
	}
	return images
}

func (t Theme) GetStaticTexts() map[string]entity.StaticTexts {
	images := make(map[string]entity.StaticTexts)

	if t.theme["static_text"] == nil {
		return images
	}

	for name, i := range t.theme["static_text"].(map[string]interface{}) {
		fmt.Printf("%s [%#v]\n", name, i)
		var bgColor color.Color
		var fColor color.Color
		var fface font.Face

		if i.(map[string]interface{})["font_color"] != nil {
			bgcolor := i.(map[string]interface{})["font_color"].(string)
			fColor = utils.ConvertToColor(bgcolor, color.White)
		} else {
			fColor = color.White
		}

		if i.(map[string]interface{})["background_color"] != nil {
			bgcolor := i.(map[string]interface{})["background_color"].(string)
			bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
		} else {
			bgColor = color.Transparent
		}

		if i.(map[string]interface{})["font"] != nil {
			fPath := fontPath + i.(map[string]interface{})["font"].(string)
			fSize := float64(i.(map[string]interface{})["font_size"].(int))
			fface = utils.LoadFontFace(fPath, fSize)
		} else {
			fface = utils.DefaultFont
		}

		images[name] = entity.StaticTexts{
			Text:            i.(map[string]interface{})["text"].(string),
			Font:            fface,
			FontColor:       fColor,
			X:               i.(map[string]interface{})["x"].(int),
			Y:               i.(map[string]interface{})["y"].(int),
			BackgroundColor: bgColor,
		}
	}
	return images
}
