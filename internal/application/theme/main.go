package theme

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"golang.org/x/image/font"
	"image/color"
	"reflect"
	"time"
)

type Theme struct {
	theme    theme.Theme
	path     string
	fontPath string
	log      *logger.Logger
}

func NewTheme(file string, l *logger.Logger) (*Theme, error) {
	var config theme.Theme

	theme.DefaultImagePath = fmt.Sprintf("res/themes/%s/", file)

	tfile := fmt.Sprintf("res/themes/%s/theme.yaml", file)

	viper.SetConfigType("yaml")
	viper.SetConfigFile(tfile)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	err = viper.Unmarshal(&config, func(config *mapstructure.DecoderConfig) {
		config.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			Hook(fmt.Sprintf("res/themes/%s/", file), "res/fonts/"),
		)
	})
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config file: %w", err)
	}

	return &Theme{
		theme:    config,
		path:     fmt.Sprintf("res/themes/%s/", file),
		fontPath: "res/fonts/",
		log:      l,
	}, nil
}

func Hook(imagePath, fontPath string) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {

		//fmt.Printf("[%+v] [%+v] [%+v]\n", f, t, data)

		if f.Kind() == reflect.Map && t == reflect.TypeOf(theme.StaticImage{}) {
			return theme.StaticImage{
				Path:   imagePath + data.(map[string]interface{})["path"].(string),
				Height: data.(map[string]interface{})["height"].(int),
				Width:  data.(map[string]interface{})["width"].(int),
				X:      data.(map[string]interface{})["x"].(int),
				Y:      data.(map[string]interface{})["y"].(int),
			}, nil
		} else if f.Kind() == reflect.Int && t == reflect.TypeOf(time.Duration(1)) {
			fmt.Printf("[%+v] [%+v] [%+v]\n", f, t, data)
			if data != nil {
				return time.Duration(data.(int)) * time.Second, nil
			} else {
				return time.Duration(0), nil
			}

		} else if f.Kind() == reflect.Map && t == reflect.TypeOf(theme.StaticText{}) {
			var bgColor color.Color
			var fColor color.Color
			var fface font.Face
			if data.(map[string]interface{})["font_color"] != nil {
				bgcolor := data.(map[string]interface{})["font_color"].(string)
				fColor = utils.ConvertToColor(bgcolor, color.White)
			} else {
				fColor = color.White
			}

			if data.(map[string]interface{})["background_color"] != nil {
				bgcolor := data.(map[string]interface{})["background_color"].(string)
				bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
			} else {
				bgColor = color.Transparent
			}

			if data.(map[string]interface{})["font"] != nil {
				fPath := fontPath + data.(map[string]interface{})["font"].(string)
				fSize := float64(data.(map[string]interface{})["font_size"].(int))
				fface = utils.LoadFontFace(fPath, fSize)
			} else {
				fface = utils.DefaultFont
			}

			return theme.StaticText{
				Text:            data.(map[string]interface{})["text"].(string),
				Font:            fface,
				FontColor:       fColor,
				X:               data.(map[string]interface{})["x"].(int),
				Y:               data.(map[string]interface{})["y"].(int),
				BackgroundColor: bgColor,
			}, nil
		} else if f.Kind() == reflect.Map && t == reflect.TypeOf(theme.Display{}) {
			return theme.Display{
				Size:        data.(map[string]interface{})["size"].(string),
				Orientation: theme.StringToOrientation(data.(map[string]interface{})["orientation"].(string)),
			}, nil
		} else if f.Kind() == reflect.Map && t == reflect.TypeOf(theme.Text{}) {
			var bgColor color.Color
			var fColor color.Color
			var fface font.Face
			var show bool
			var showUnit bool
			var bgImage string
			if data.(map[string]interface{})["font_color"] != nil {
				bgcolor := data.(map[string]interface{})["font_color"].(string)
				fColor = utils.ConvertToColor(bgcolor, color.White)
			} else {
				fColor = color.White
			}
			if data.(map[string]interface{})["background_image"] != nil {
				bgImage = imagePath + data.(map[string]interface{})["background_image"].(string)
			} else {
				bgImage = ""
			}

			if data.(map[string]interface{})["font"] != nil {
				fPath := fontPath + data.(map[string]interface{})["font"].(string)
				fSize := float64(data.(map[string]interface{})["font_size"].(int))
				fface = utils.LoadFontFace(fPath, fSize)
			} else {
				fface = utils.DefaultFont
			}

			if data.(map[string]interface{})["background_color"] != nil {
				bgcolor := data.(map[string]interface{})["background_color"].(string)
				bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
			} else {
				bgColor = color.Transparent
			}

			if data.(map[string]interface{})["show_unit"] != nil {
				showUnit = data.(map[string]interface{})["show_unit"].(bool)
			} else {
				showUnit = false
			}
			if data.(map[string]interface{})["show"] != nil {
				show = data.(map[string]interface{})["show"].(bool)
			} else {
				show = false
			}
			return theme.Text{
				Show:            show,
				ShowUnit:        showUnit,
				BackgroundImage: bgImage,
				Font:            fface,
				FontColor:       fColor,
				Align:           theme.LEFT,
				Padding:         4,
				X:               data.(map[string]interface{})["x"].(int),
				Y:               data.(map[string]interface{})["y"].(int),
				BackgroundColor: bgColor,
			}, nil
		} else if f.Kind() == reflect.Map && t == reflect.TypeOf(theme.Graph{}) {
			var bgColor color.Color
			var show bool
			var bgImage string

			if data.(map[string]interface{})["show"] != nil {
				show = data.(map[string]interface{})["show"].(bool)
			} else {
				show = false
			}
			if data.(map[string]interface{})["bar_color"] != nil {
				bgcolor := data.(map[string]interface{})["bar_color"].(string)
				bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
			} else {
				bgColor = color.Transparent
			}
			if data.(map[string]interface{})["background_image"] != nil {
				bgImage = imagePath + data.(map[string]interface{})["background_image"].(string)
			} else {
				bgImage = ""
			}
			return theme.Graph{
				Show:            show,
				X:               data.(map[string]interface{})["x"].(int),
				Y:               data.(map[string]interface{})["y"].(int),
				Width:           data.(map[string]interface{})["width"].(int),
				Height:          data.(map[string]interface{})["height"].(int),
				MinValue:        data.(map[string]interface{})["min_value"].(int),
				MaxValue:        data.(map[string]interface{})["max_value"].(int),
				BarColor:        bgColor,
				BarOutline:      data.(map[string]interface{})["bar_outline"].(bool),
				BackgroundImage: bgImage,
			}, nil
		} else if f.Kind() == reflect.Map && t == reflect.TypeOf(theme.Radial{}) {
			var bgColor color.Color
			var fColor color.Color
			var fface font.Face
			var show bool
			var showText bool
			var showUnit bool
			var bgImage string

			if data.(map[string]interface{})["bar_color"] != nil {
				bgcolor := data.(map[string]interface{})["bar_color"].(string)
				bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
			} else {
				bgColor = color.Transparent
			}

			if data.(map[string]interface{})["background_image"] != nil {
				bgImage = imagePath + data.(map[string]interface{})["background_image"].(string)
			} else {
				bgImage = ""
			}

			if data.(map[string]interface{})["background_color"] != nil {
				bgcolor := data.(map[string]interface{})["background_color"].(string)
				bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
			} else {
				bgColor = color.Transparent
			}

			if data.(map[string]interface{})["font_color"] != nil {
				bgcolor := data.(map[string]interface{})["font_color"].(string)
				fColor = utils.ConvertToColor(bgcolor, color.Transparent)
			} else {
				fColor = color.Transparent
			}

			if data.(map[string]interface{})["show_unit"] != nil {
				showUnit = data.(map[string]interface{})["show_unit"].(bool)
			} else {
				showUnit = false
			}
			if data.(map[string]interface{})["show"] != nil {
				show = data.(map[string]interface{})["show"].(bool)
			} else {
				show = false
			}
			if data.(map[string]interface{})["show_text"] != nil {
				showText = data.(map[string]interface{})["show_text"].(bool)
			} else {
				showText = false
			}

			if data.(map[string]interface{})["font"] != nil {
				fPath := fontPath + data.(map[string]interface{})["font"].(string)
				fSize := float64(data.(map[string]interface{})["font_size"].(int))
				fface = utils.LoadFontFace(fPath, fSize)
			} else {
				fface = utils.DefaultFont
			}

			return theme.Radial{
				Show:            show,
				X:               data.(map[string]interface{})["x"].(int),
				Y:               data.(map[string]interface{})["y"].(int),
				Radius:          data.(map[string]interface{})["radius"].(int),
				Width:           data.(map[string]interface{})["width"].(int),
				MinValue:        data.(map[string]interface{})["min_value"].(int),
				MaxValue:        data.(map[string]interface{})["max_value"].(int),
				AngleStart:      data.(map[string]interface{})["angle_start"].(int),
				AngleEnd:        data.(map[string]interface{})["angle_end"].(int),
				AngleSteps:      data.(map[string]interface{})["angle_steps"].(int),
				AngleStep:       data.(map[string]interface{})["angle_step"].(int),
				Clockwise:       data.(map[string]interface{})["clockwise"].(bool),
				BarColor:        bgColor,
				ShowText:        showText,
				ShowUnit:        showUnit,
				Font:            fface,
				FontColor:       fColor,
				BackgroundColor: bgColor,
				BackgroundImage: bgImage,
			}, nil
		}
		return data, nil
	}
}

func (t *Theme) GetStaticImages() map[string]theme.StaticImage {
	return t.theme.StaticImages
}

func (t *Theme) GetStaticTexts() map[string]theme.StaticText {
	return t.theme.StaticTexts
}

func (t *Theme) GetStats() *theme.Stats {
	return t.theme.Stats
}

func (t *Theme) GetThemePath() string {
	return t.path
}
