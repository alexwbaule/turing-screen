package theme

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/config"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"golang.org/x/image/font"
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

func NewTheme(cfg *config.Config, l *logger.Logger) (*Theme, error) {
	var cconfig theme.Theme
	file := cfg.GetThemeName()
	reverse := cfg.GetDeviceDisplay().Reverse

	theme.DefaultImagePath = fmt.Sprintf("res/themes/%s/", file)

	tfile := fmt.Sprintf("res/themes/%s/theme.yaml", file)

	l.Infof("Loading theme from: %s", tfile)

	viper.SetConfigType("yaml")
	viper.SetConfigFile(tfile)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	err = viper.Unmarshal(&cconfig, func(config *mapstructure.DecoderConfig) {
		config.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			Hook(file, reverse),
		)
	})
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config file: %w", err)
	}

	applyDefaults(&cconfig, l)

	if cconfig.Display.Size != "5\"" {
		l.Fatalf("this theme is not for this device [%s -> %s]", cconfig.Display.Size, cconfig.Display.Orientation)
	}

	return &Theme{
		theme:    cconfig,
		path:     fmt.Sprintf("res/themes/%s/", file),
		fontPath: "res/fonts/",
		log:      l,
	}, nil
}

// applyDefaultBackground aplica um ponteiro para a imagem de fundo padrão a um BackgroundStyle
func applyDefaultBackground(style *theme.BackgroundStyle, defaultBg *image.Image, defaultBgPath string) {
	if style != nil && style.BackgroundImage == nil {
		style.BackgroundImage = *defaultBg
		style.BackgroundImagePath = defaultBgPath
	}
}

func applyDefaults(cconfig *theme.Theme, l *logger.Logger) {
	// Lógica de herança de fundo global
	if bg, ok := cconfig.StaticImages["background"]; ok && bg.Path != "" {
		defaultBgImage, err := utils.LoadImage(bg.Path)
		if err != nil {
			l.Errorf("failed to load default background image: %s", err)
		} else {
			stats := cconfig.Stats
			if stats != nil {
				// Função auxiliar para aplicar o fundo a um Mesurement
				applyToMesurement := func(m *theme.Mesurement) {
					if m == nil {
						return
					}
					if m.Graph != nil {
						applyDefaultBackground(&m.Graph.BackgroundStyle, &defaultBgImage, bg.Path)
					}
					if m.Radial != nil {
						applyDefaultBackground(&m.Radial.BackgroundStyle, &defaultBgImage, bg.Path)
					}
					if m.Text != nil {
						applyDefaultBackground(&m.Text.BackgroundStyle, &defaultBgImage, bg.Path)
					}
					if m.Percent != nil {
						applyDefaultBackground(&m.Percent.BackgroundStyle, &defaultBgImage, bg.Path)
					}
				}

				// CPU
				if stats.CPU != nil {
					applyToMesurement(stats.CPU.Percentage)
					applyToMesurement(stats.CPU.Frequency)
					applyToMesurement(stats.CPU.Temperature)
					if stats.CPU.Load != nil {
						if stats.CPU.Load.One != nil && stats.CPU.Load.One.Text != nil {
							applyDefaultBackground(&stats.CPU.Load.One.Text.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.CPU.Load.Five != nil && stats.CPU.Load.Five.Text != nil {
							applyDefaultBackground(&stats.CPU.Load.Five.Text.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.CPU.Load.Fifteen != nil && stats.CPU.Load.Fifteen.Text != nil {
							applyDefaultBackground(&stats.CPU.Load.Fifteen.Text.BackgroundStyle, &defaultBgImage, bg.Path)
						}
					}
				}
				// GPU
				if stats.GPU != nil {
					applyToMesurement(stats.GPU.Percentage)
					applyToMesurement(stats.GPU.Temperature)
					applyToMesurement(stats.GPU.Power)
					applyToMesurement(stats.GPU.Memory)
				}
				// Memory
				if stats.Memory != nil {
					if stats.Memory.Swap != nil {
						if stats.Memory.Swap.Graph != nil {
							applyDefaultBackground(&stats.Memory.Swap.Graph.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Memory.Swap.Radial != nil {
							applyDefaultBackground(&stats.Memory.Swap.Radial.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Memory.Swap.Used != nil {
							applyDefaultBackground(&stats.Memory.Swap.Used.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Memory.Swap.Free != nil {
							applyDefaultBackground(&stats.Memory.Swap.Free.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Memory.Swap.PercentText != nil {
							applyDefaultBackground(&stats.Memory.Swap.PercentText.BackgroundStyle, &defaultBgImage, bg.Path)
						}
					}
					if stats.Memory.Virtual != nil {
						if stats.Memory.Virtual.Graph != nil {
							applyDefaultBackground(&stats.Memory.Virtual.Graph.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Memory.Virtual.Radial != nil {
							applyDefaultBackground(&stats.Memory.Virtual.Radial.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Memory.Virtual.Used != nil {
							applyDefaultBackground(&stats.Memory.Virtual.Used.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Memory.Virtual.Free != nil {
							applyDefaultBackground(&stats.Memory.Virtual.Free.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Memory.Virtual.PercentText != nil {
							applyDefaultBackground(&stats.Memory.Virtual.PercentText.BackgroundStyle, &defaultBgImage, bg.Path)
						}
					}
				}
				// Disk
				if stats.Disk != nil {
					applyToMesurement(stats.Disk.Used)
					applyToMesurement(stats.Disk.Total)
					applyToMesurement(stats.Disk.Free)
					applyToMesurement(stats.Disk.Temperature)
				}
				// Net
				if stats.Net != nil {
					applyToNetwork := func(n *theme.NetworkMesurement) {
						if n == nil {
							return
						}
						if n.Upload != nil && n.Upload.Text != nil {
							applyDefaultBackground(&n.Upload.Text.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if n.Download != nil && n.Download.Text != nil {
							applyDefaultBackground(&n.Download.Text.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if n.Uploaded != nil && n.Uploaded.Text != nil {
							applyDefaultBackground(&n.Uploaded.Text.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if n.Downloaded != nil && n.Downloaded.Text != nil {
							applyDefaultBackground(&n.Downloaded.Text.BackgroundStyle, &defaultBgImage, bg.Path)
						}
					}
					applyToNetwork(stats.Net.Wired)
					applyToNetwork(stats.Net.Wifi)
				}
				// Date
				if stats.Date != nil {
					if stats.Date.Day != nil && stats.Date.Day.Text != nil {
						applyDefaultBackground(&stats.Date.Day.Text.BackgroundStyle, &defaultBgImage, bg.Path)
					}
					if stats.Date.Hour != nil && stats.Date.Hour.Text != nil {
						applyDefaultBackground(&stats.Date.Hour.Text.BackgroundStyle, &defaultBgImage, bg.Path)
					}
				}
				// Weather
				if stats.Weather != nil {
					if stats.Weather.Condition != nil {
						applyDefaultBackground(&stats.Weather.Condition.BackgroundStyle, &defaultBgImage, bg.Path)
					}
					if stats.Weather.Temperature != nil {
						if stats.Weather.Temperature.Text != nil {
							applyDefaultBackground(&stats.Weather.Temperature.Text.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Weather.Temperature.Graph != nil {
							applyDefaultBackground(&stats.Weather.Temperature.Graph.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Weather.Temperature.Radial != nil {
							applyDefaultBackground(&stats.Weather.Temperature.Radial.BackgroundStyle, &defaultBgImage, bg.Path)
						}
						if stats.Weather.Temperature.Percent != nil {
							applyDefaultBackground(&stats.Weather.Temperature.Percent.BackgroundStyle, &defaultBgImage, bg.Path)
						}
					}
				}
			}

			// StaticTexts
			for key, st := range cconfig.StaticTexts {
				applyDefaultBackground(&st.BackgroundStyle, &defaultBgImage, bg.Path)
				cconfig.StaticTexts[key] = st
			}
			if cconfig.VideoPlay != nil && cconfig.VideoPlay.ForegroundImage == image.Transparent {
				cconfig.VideoPlay.ForegroundImage = defaultBgImage
			}
		}
	}
}

func Hook(file string, reverse bool) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {

		if f.Kind() == reflect.Map && (t == reflect.TypeOf(&theme.Text{}) || t == reflect.TypeOf(&theme.Text{})) {
			return translateText(file, data.(map[string]interface{}))
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.StaticText{}) || t == reflect.TypeOf(&theme.StaticText{})) {
			return translateStaticText(file, data.(map[string]interface{}))
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.Display{}) || t == reflect.TypeOf(&theme.Display{})) {
			return translateDisplay(data.(map[string]interface{}), reverse)
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.StaticImage{}) || t == reflect.TypeOf(&theme.StaticImage{})) {
			return translateStaticImage(file, data.(map[string]interface{}))
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.VideoPlay{}) || t == reflect.TypeOf(&theme.VideoPlay{})) {
			return translateVideoPlay(file, data.(map[string]interface{}))
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

func parseLayout(data map[string]interface{}) theme.Layout {
	x := 0
	if val, ok := data["x"].(int); ok {
		x = val
	}

	y := 0
	if val, ok := data["y"].(int); ok {
		y = val
	}

	width := 0
	if val, ok := data["width"].(int); ok {
		width = val
	}

	height := 0
	if val, ok := data["height"].(int); ok {
		height = val
	}

	return theme.Layout{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}

func parseBackgroundStyle(data map[string]interface{}, file string) (theme.BackgroundStyle, error) {
	var bgColor color.Color
	var bgImagePath string
	var bgImage image.Image
	var err error
	imagePath := fmt.Sprintf(THEMEPATH, file)

	if val, ok := data["background_color"].(string); ok {
		bgColor = utils.ConvertToColor(val, color.Transparent)
	} else {
		bgColor = color.Transparent
	}

	if val, ok := data["background_image"].(string); ok {
		bgImagePath = imagePath + val
		bgImage, err = utils.LoadImage(bgImagePath)
		if err != nil {
			return theme.BackgroundStyle{}, err
		}

	}

	return theme.BackgroundStyle{
		BackgroundColor:     bgColor,
		BackgroundImagePath: bgImagePath,
		BackgroundImage:     bgImage,
	}, nil
}

func parseTextStyle(data map[string]interface{}) theme.TextStyle {
	var fColor color.Color
	var fface font.Face
	var align theme.Alignment

	if val, ok := data["font_color"].(string); ok {
		fColor = utils.ConvertToColor(val, color.White)
	} else {
		fColor = color.White
	}

	if val, ok := data["font"].(string); ok {
		if size, ok := data["font_size"].(int); ok {
			fPath := FONTPATH + val
			fface = utils.LoadFontFace(fPath, float64(size))
		} else {
			fface = utils.DefaultFont
		}
	} else {
		fface = utils.DefaultFont
	}

	if val, ok := data["align"].(string); ok {
		align = theme.StringToAlignment(val)
	} else {
		align = theme.LEFT
	}

	return theme.TextStyle{
		Font:      fface,
		FontColor: fColor,
		Align:     align,
	}
}

func translateDuration(data interface{}) (interface{}, error) {
	if val, ok := data.(int); ok && val > 0 {
		return time.Duration(val) * time.Second, nil
	}
	return time.Duration(1) * time.Second, nil
}

func translateGraph(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	bgStyle, err := parseBackgroundStyle(data, file)
	if err != nil {
		return nil, err
	}

	var barColor color.Color
	if val, ok := data["bar_color"].(string); ok {
		barColor = utils.ConvertToColor(val, color.Transparent)
	} else {
		barColor = color.Transparent
	}

	show := false
	if val, ok := data["show"].(bool); ok {
		show = val
	}

	v := theme.Graph{
		Layout:          layout,
		BackgroundStyle: bgStyle,
		Show:            show,
		MinValue:        data["min_value"].(int),
		MaxValue:        data["max_value"].(int),
		BarColor:        barColor,
		BarOutline:      data["bar_outline"].(bool),
	}
	return v, nil
}

func translateRadial(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	bgStyle, err := parseBackgroundStyle(data, file)
	if err != nil {
		return nil, err
	}
	textStyle := parseTextStyle(data)

	var barColor color.Color
	if val, ok := data["bar_color"].(string); ok {
		barColor = utils.ConvertToColor(val, color.Transparent)
	} else {
		barColor = color.Transparent
	}

	show := false
	if val, ok := data["show"].(bool); ok {
		show = val
	}

	showText := false
	if val, ok := data["show_text"].(bool); ok {
		showText = val
	}

	showUnit := false
	if val, ok := data["show_unit"].(bool); ok {
		showUnit = val
	}

	v := theme.Radial{
		Layout:          layout,
		BackgroundStyle: bgStyle,
		TextStyle:       textStyle,
		Show:            show,
		Radius:          data["radius"].(int),
		MinValue:        data["min_value"].(int),
		MaxValue:        data["max_value"].(int),
		AngleStart:      data["angle_start"].(int),
		AngleEnd:        data["angle_end"].(int),
		AngleSteps:      data["angle_steps"].(int),
		AngleSep:        data["angle_sep"].(int),
		Clockwise:       data["clockwise"].(bool),
		BarColor:        barColor,
		ShowText:        showText,
		ShowUnit:        showUnit,
	}
	return v, nil
}

func translateDisplay(data map[string]interface{}, reverse bool) (interface{}, error) {
	size := "empty"
	if val, ok := data["size"].(string); ok {
		size = val
	}
	orientation, ok := data["orientation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing orientation on theme")
	}
	v := theme.Display{
		Size:        size,
		Orientation: theme.StringToOrientation(orientation, reverse),
	}
	return v, nil
}

func translateStaticImage(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	imagePath := fmt.Sprintf(THEMEPATH, file)

	if val, ok := data["path"].(string); ok {
		bgImagePath := imagePath + val
		bgImage, err := utils.LoadImage(bgImagePath)
		if err != nil {
			return theme.BackgroundStyle{}, err
		}
		fmt.Printf("Bounds: %#v\n", bgImage.Bounds())

		return theme.StaticImage{
			Layout:          layout,
			BackgroundImage: bgImage,
			Path:            imagePath + data["path"].(string),
		}, nil
	}
	return nil, fmt.Errorf("missing required config path on static_images")
}

func translateVideoPlay(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	imagePath := fmt.Sprintf(THEMEPATH, file)

	if val, ok := data["path"].(string); ok {
		bgImagePath := imagePath + val
		bVideo, vSize, err := utils.LoadVideo(bgImagePath)
		if err != nil {
			return theme.BackgroundStyle{}, err
		}
		return theme.VideoPlay{
			Layout:          layout,
			Video:           bVideo,
			Size:            vSize,
			ForegroundImage: image.Transparent,
			Path:            imagePath + data["path"].(string),
			DevicePath:      "/root/video/" + data["path"].(string),
		}, nil
	}
	return nil, fmt.Errorf("missing required config path on static_images")
}

func translateStaticText(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	bgStyle, err := parseBackgroundStyle(data, file)
	if err != nil {
		return nil, err
	}
	textStyle := parseTextStyle(data)

	v := theme.StaticText{
		Layout:          layout,
		TextStyle:       textStyle,
		BackgroundStyle: bgStyle,
		Text:            data["text"].(string),
	}
	return v, nil
}

func translateText(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	bgStyle, err := parseBackgroundStyle(data, file)
	if err != nil {
		return nil, err
	}
	textStyle := parseTextStyle(data)

	show := false
	if val, ok := data["show"].(bool); ok {
		show = val
	}

	showUnit := false
	if val, ok := data["show_unit"].(bool); ok {
		showUnit = val
	}

	var format theme.Format
	if val, ok := data["format"].(string); ok {
		format = theme.StringToFormat(val)
	} else {
		format = theme.SHORT
	}

	v := theme.Text{
		Layout:          layout,
		BackgroundStyle: bgStyle,
		TextStyle:       textStyle,
		Show:            show,
		ShowUnit:        showUnit,
		Format:          format,
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
func (t *Theme) GetDisplay() *theme.Display {
	return t.theme.Display
}

func (t *Theme) GetPath() string {
	return t.path
}

func (t *Theme) GetVideoPlay() *theme.VideoPlay {
	return t.theme.VideoPlay
}
