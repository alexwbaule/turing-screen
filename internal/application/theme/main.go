package theme

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/config"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
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

	for b, i := range t.theme["static_text"].(map[string]interface{}) {
		fmt.Printf("%s [%#v]\n", b, i)
		var bgtype entity.BackgroundType
		var background string

		if i.(map[string]interface{})["background_image"] != nil {
			bgtype = entity.IMAGE
			background = i.(map[string]interface{})["background_image"].(string)
		} else if i.(map[string]interface{})["background_color"] != nil {
			bgtype = entity.COLOR
			background = i.(map[string]interface{})["background_color"].(string)
		}

		images[b] = entity.StaticTexts{
			Text:           i.(map[string]interface{})["text"].(string),
			Font:           fontPath + i.(map[string]interface{})["font"].(string),
			FontSize:       i.(map[string]interface{})["font_size"].(int),
			FontColor:      i.(map[string]interface{})["font_color"].(string),
			X:              i.(map[string]interface{})["x"].(int),
			Y:              i.(map[string]interface{})["y"].(int),
			BackgroundType: bgtype,
			Background:     background,
		}
	}
	return images
}
