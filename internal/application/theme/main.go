package theme

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"golang.org/x/image/font"
	"image"
	"image/color"
	"reflect"
	"time"
)

const (
	THEMEPATH = "res/themes/%s/"
	FONTPATH  = "res/fonts/"
)

type Duration time.Duration

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

	l.Infof("Loading theme from: %s", tfile)

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
			Hook(file),
		)
	})
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config file: %w", err)
	}

	fmt.Printf("Display:[%#v]\n", config.Display)
	fmt.Printf("StaticTexts:[%#v]\n", config.StaticTexts)
	fmt.Printf("StaticImages:[%#v]\n", config.StaticImages)
	fmt.Printf("CPU:[%#v]\n", config.Stats.CPU)
	fmt.Printf("GPU:[%#v]\n", config.Stats.GPU)
	fmt.Printf("MEMORY:[%#v]\n", config.Stats.Memory)
	fmt.Printf("DISK:[%#v]\n", config.Stats.Disk)
	fmt.Printf("DATE:[%#v]\n", config.Stats.Date)
	fmt.Printf("NET:[%#v]\n", config.Stats.Net.Wired)
	fmt.Printf("NET:[%#v]\n", config.Stats.Net.Wifi)

	return &Theme{
		theme:    config,
		path:     fmt.Sprintf("res/themes/%s/", file),
		fontPath: "res/fonts/",
		log:      l,
	}, nil
}

func Hook(file string) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {

		if f.Kind() == reflect.Map && (t == reflect.TypeOf(&theme.Text{}) || t == reflect.TypeOf(&theme.Text{})) {
			return translateText(file, data.(map[string]interface{}))
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.StaticText{}) || t == reflect.TypeOf(&theme.StaticText{})) {
			return translateStaticText(data.(map[string]interface{}))
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.Display{}) || t == reflect.TypeOf(&theme.Display{})) {
			return translateDisplay(data.(map[string]interface{}))
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.StaticImage{}) || t == reflect.TypeOf(&theme.StaticImage{})) {
			return translateStaticImage(file, data.(map[string]interface{}))
		}
		if t == reflect.TypeOf(time.Duration(1)) {
			return translateDuration(data.(interface{}))
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.Graph{}) || t == reflect.TypeOf(&theme.Graph{})) {
			return translateGraph(file, data.(map[string]interface{}))
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.Radial{}) || t == reflect.TypeOf(&theme.Radial{})) {
			return translateRadial(file, data.(map[string]interface{}))
		}
		return data, nil
	}
}
func translateDuration(data interface{}) (interface{}, error) {
	var v time.Duration

	if data != nil {
		v = time.Duration(data.(int)) * time.Second
	} else {
		v = time.Duration(0)
	}
	return v, nil
}

func translateGraph(file string, data map[string]interface{}) (interface{}, error) {
	var bgColor color.Color
	var show bool
	var bgImagePath string
	var bgImage image.Image
	var err error
	imagePath := fmt.Sprintf(THEMEPATH, file)

	if data["show"] != nil {
		show = data["show"].(bool)
	} else {
		show = false
	}
	if data["bar_color"] != nil {
		bgcolor := data["bar_color"].(string)
		bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
	} else {
		bgColor = color.Transparent
	}
	if data["background_image"] != nil {
		bgImagePath = imagePath + data["background_image"].(string)
		bgImage, err = utils.LoadImage(bgImagePath)
		if err != nil {
			return nil, err
		}
	} else {
		bgImagePath = ""
		bgImage = nil
	}
	v := theme.Graph{
		Show:                show,
		X:                   data["x"].(int),
		Y:                   data["y"].(int),
		Width:               data["width"].(int),
		Height:              data["height"].(int),
		MinValue:            data["min_value"].(int),
		MaxValue:            data["max_value"].(int),
		BarColor:            bgColor,
		BarOutline:          data["bar_outline"].(bool),
		BackgroundImage:     bgImage,
		BackgroundImagePath: bgImagePath,
	}
	return v, nil
}
func translateRadial(file string, data map[string]interface{}) (interface{}, error) {
	var bgColor color.Color
	var fColor color.Color
	var fface font.Face
	var show bool
	var showText bool
	var showUnit bool
	var bgImagePath string
	var bgImage image.Image
	var err error
	imagePath := fmt.Sprintf(THEMEPATH, file)

	if data["bar_color"] != nil {
		bgcolor := data["bar_color"].(string)
		bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
	} else {
		bgColor = color.Transparent
	}

	if data["background_image"] != nil {
		bgImagePath = imagePath + data["background_image"].(string)
		bgImage, err = utils.LoadImage(bgImagePath)
		if err != nil {
			return nil, err
		}
	} else {
		bgImagePath = ""
		bgImage = nil
	}

	if data["background_color"] != nil {
		bgcolor := data["background_color"].(string)
		bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
	} else {
		bgColor = color.Transparent
	}

	if data["font_color"] != nil {
		bgcolor := data["font_color"].(string)
		fColor = utils.ConvertToColor(bgcolor, color.Transparent)
	} else {
		fColor = color.Transparent
	}

	if data["show_unit"] != nil {
		showUnit = data["show_unit"].(bool)
	} else {
		showUnit = false
	}
	if data["show"] != nil {
		show = data["show"].(bool)
	} else {
		show = false
	}
	if data["show_text"] != nil {
		showText = data["show_text"].(bool)
	} else {
		showText = false
	}

	if data["font"] != nil {
		fPath := FONTPATH + data["font"].(string)
		fSize := float64(data["font_size"].(int))
		fface = utils.LoadFontFace(fPath, fSize)
	} else {
		fface = utils.DefaultFont
	}

	v := theme.Radial{
		Show:                show,
		X:                   data["x"].(int),
		Y:                   data["y"].(int),
		Radius:              data["radius"].(int),
		Width:               data["width"].(int),
		MinValue:            data["min_value"].(int),
		MaxValue:            data["max_value"].(int),
		AngleStart:          data["angle_start"].(int),
		AngleEnd:            data["angle_end"].(int),
		AngleSteps:          data["angle_steps"].(int),
		AngleStep:           data["angle_step"].(int),
		Clockwise:           data["clockwise"].(bool),
		BarColor:            bgColor,
		ShowText:            showText,
		ShowUnit:            showUnit,
		Font:                fface,
		FontColor:           fColor,
		BackgroundColor:     bgColor,
		BackgroundImage:     bgImage,
		BackgroundImagePath: bgImagePath,
	}
	return v, nil
}

func translateDisplay(data map[string]interface{}) (interface{}, error) {
	v := theme.Display{
		Size:        data["size"].(string),
		Orientation: theme.StringToOrientation(data["orientation"].(string)),
	}
	return v, nil
}

func translateStaticImage(file string, data map[string]interface{}) (interface{}, error) {
	imagePath := fmt.Sprintf(THEMEPATH, file)
	v := theme.StaticImage{
		Path:   imagePath + data["path"].(string),
		Height: data["height"].(int),
		Width:  data["width"].(int),
		X:      data["x"].(int),
		Y:      data["y"].(int),
	}
	return v, nil
}

func translateStaticText(data map[string]interface{}) (interface{}, error) {
	var bgColor color.Color
	var fColor color.Color
	var fface font.Face

	if data["font_color"] != nil {
		bgcolor := data["font_color"].(string)
		fColor = utils.ConvertToColor(bgcolor, color.White)
	} else {
		fColor = color.White
	}

	if data["background_color"] != nil {
		bgcolor := data["background_color"].(string)
		bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
	} else {
		bgColor = color.Transparent
	}

	if data["font"] != nil {
		fPath := FONTPATH + data["font"].(string)
		fSize := float64(data["font_size"].(int))
		fface = utils.LoadFontFace(fPath, fSize)
	} else {
		fface = utils.DefaultFont
	}

	v := theme.StaticText{
		Text:            data["text"].(string),
		Font:            fface,
		FontColor:       fColor,
		X:               data["x"].(int),
		Y:               data["y"].(int),
		BackgroundColor: bgColor,
	}
	return v, nil
}

func translateText(file string, data map[string]interface{}) (interface{}, error) {
	imagePath := fmt.Sprintf(THEMEPATH, file)
	var bgColor color.Color
	var fColor color.Color
	var fface font.Face
	var show bool
	var showUnit bool
	var bgImagePath string
	var bgImage image.Image
	var align theme.Alignment
	var format theme.Format

	var err error

	if data["font_color"] != nil {
		bgcolor := data["font_color"].(string)
		fColor = utils.ConvertToColor(bgcolor, color.White)
	} else {
		fColor = color.White
	}
	if data["background_image"] != nil {
		bgImagePath = imagePath + data["background_image"].(string)
		bgImage, err = utils.LoadImage(bgImagePath)
		if err != nil {
			return nil, err
		}
	} else {
		bgImagePath = ""
		bgImage = nil
	}

	if data["font"] != nil {
		fPath := FONTPATH + data["font"].(string)
		fSize := float64(data["font_size"].(int))
		fface = utils.LoadFontFace(fPath, fSize)
	} else {
		fface = utils.DefaultFont
	}

	if data["background_color"] != nil {
		bgcolor := data["background_color"].(string)
		bgColor = utils.ConvertToColor(bgcolor, color.Transparent)
	} else {
		bgColor = color.Transparent
	}

	if data["show_unit"] != nil {
		showUnit = data["show_unit"].(bool)
	} else {
		showUnit = false
	}

	if data["format"] != nil {
		v := data["format"].(string)
		format = theme.StringToFormat(v)
	} else {
		format = theme.SHORT
	}

	if data["show"] != nil {
		show = data["show"].(bool)
	} else {
		show = false
	}

	if data["align"] != nil {
		v := data["align"].(string)
		align = theme.StringToAlignment(v)
	} else {
		align = theme.LEFT
	}
	v := theme.Text{
		Show:                show,
		ShowUnit:            showUnit,
		Format:              format,
		BackgroundImage:     bgImage,
		BackgroundImagePath: bgImagePath,
		Font:                fface,
		FontColor:           fColor,
		Align:               align,
		X:                   data["x"].(int),
		Y:                   data["y"].(int),
		BackgroundColor:     bgColor,
	}
	return v, nil
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
