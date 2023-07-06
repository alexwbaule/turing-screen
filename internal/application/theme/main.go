package theme

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
	"github.com/spf13/viper"
	"golang.org/x/image/font"
	"image/color"
	"reflect"
	"strings"
	"time"
)

const (
	display       = "display"
	static_images = "static_images"
	static_texts  = "static_texts"
	stats         = "stats"
)

type Theme struct {
	theme       map[string]interface{}
	orientation entity.Orientation
	path        string
}

const fontPath = "res/fonts/"

func newConfig(file string) (*viper.Viper, error) {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	v.SetConfigType("yaml")
	v.SetConfigFile(file)
	f := v.ReadInConfig()
	return v, f
}

func LoadTheme(themeFile string) (*Theme, error) {
	tfile := fmt.Sprintf("res/themes/%s/theme.yaml", themeFile)

	cfg, err := newConfig(tfile)
	if err != nil {
		return nil, err
	}

	theme := cfg.AllSettings()

	if theme[display] == nil {
		return nil, fmt.Errorf("missing display configuration")
	}
	if theme[static_images] == nil {
		return nil, fmt.Errorf("missing static_images configuration")
	}
	if theme[static_texts] == nil {
		return nil, fmt.Errorf("missing static_texts configuration")
	}
	if theme[stats] == nil {
		return nil, fmt.Errorf("missing stats configuration")
	}

	return &Theme{
		theme:       cfg.AllSettings(),
		path:        fmt.Sprintf("res/themes/%s/", themeFile),
		orientation: entity.StringToOrientation(cfg.GetString("display.orientation")),
	}, nil
}

func (t Theme) GetOrientation() entity.Orientation {
	return t.orientation
}

func (t Theme) GetStaticImages() map[string]entity.StaticImage {
	images := make(map[string]entity.StaticImage)

	if t.theme[static_images] == nil {
		return images
	}

	for b, i := range t.theme[static_images].(map[string]interface{}) {
		images[b] = entity.StaticImage{
			Height: i.(map[string]interface{})["height"].(int),
			Path:   t.path + i.(map[string]interface{})["path"].(string),
			Width:  i.(map[string]interface{})["width"].(int),
			X:      i.(map[string]interface{})["x"].(int),
			Y:      i.(map[string]interface{})["y"].(int),
		}
	}
	return images
}

func (t Theme) GetStaticTexts() map[string]entity.StaticText {
	images := make(map[string]entity.StaticText)

	if t.theme[static_texts] == nil {
		return images
	}

	for name, i := range t.theme[static_texts].(map[string]interface{}) {
		var bgColor color.Color
		var fColor color.Color
		var fface font.Face

		//fmt.Printf("%s -> %#v\n", name, i)

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

		images[name] = entity.StaticText{
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

func (t Theme) GetCPUStats() map[string]entity.CPU {
	cpu := make(map[string]entity.CPU)

	if t.theme[stats] == nil {
		return cpu
	}
	stat := t.theme[stats].(map[string]interface{})["cpu"]

	if stat == nil {
		return cpu
	}
	for name, i := range stat.(map[string]interface{}) {
		var interval int
		statTexts := make(map[string]entity.StatText)
		StatProgressBars := make(map[string]entity.StatProgressBar)
		StatRadialBars := make(map[string]entity.StatRadialBar)

		//fmt.Printf("[%s] - [%#v]\n", name, i)

		switch reflect.TypeOf(i).Kind() {
		case reflect.Int:
			if i == nil {
				interval = 1.0
			} else {
				interval = i.(int)
			}
		case reflect.Map:
			cpu[name] = entity.CPU{
				Interval:         time.Duration(interval) * time.Second,
				StatTexts:        statTexts,
				StatProgressBars: StatProgressBars,
				StatRadialBars:   StatRadialBars,
			}
		}
	}
	return nil
}

func (t Theme) GetGPUStats() map[string]entity.GPU {
	gpu := make(map[string]entity.GPU)

	if t.theme[stats] == nil {
		return gpu
	}
	stat := t.theme[stats].(map[string]interface{})["gpu"]

	if stat == nil {
		return gpu
	}
	for name, i := range stat.(map[string]interface{}) {
		var interval int
		statTexts := make(map[string]entity.StatText)
		StatProgressBars := make(map[string]entity.StatProgressBar)
		StatRadialBars := make(map[string]entity.StatRadialBar)

		//fmt.Printf("[%s] - [%#v]\n", name, i)

		switch reflect.TypeOf(i).Kind() {
		case reflect.Int:
			if i == nil {
				interval = 1.0
			} else {
				interval = i.(int)
			}
		case reflect.Map:
			gpu[name] = entity.GPU{
				Interval:         time.Duration(interval) * time.Second,
				StatTexts:        statTexts,
				StatProgressBars: StatProgressBars,
				StatRadialBars:   StatRadialBars,
			}
		}

	}
	return nil
}

func (t Theme) GetDiskStats() map[string]entity.Disk {
	disk := make(map[string]entity.Disk)

	if t.theme[stats] == nil {
		return disk
	}
	stat := t.theme[stats].(map[string]interface{})["disk"]

	if stat == nil {
		return disk
	}
	for name, i := range stat.(map[string]interface{}) {
		var interval int
		statTexts := make(map[string]entity.StatText)
		StatProgressBars := make(map[string]entity.StatProgressBar)
		StatRadialBars := make(map[string]entity.StatRadialBar)

		//fmt.Printf("[%s] - [%#v]\n", name, i)

		switch reflect.TypeOf(i).Kind() {
		case reflect.Int:
			if i == nil {
				interval = 1.0
			} else {
				interval = i.(int)
			}
		case reflect.Map:
			disk[name] = entity.Disk{
				Interval:         time.Duration(interval) * time.Second,
				StatTexts:        statTexts,
				StatProgressBars: StatProgressBars,
				StatRadialBars:   StatRadialBars,
			}
		}
	}
	return nil
}
func (t Theme) GetMemoryStats() map[string]entity.Memory {
	memory := make(map[string]entity.Memory)

	if t.theme[stats] == nil {
		return memory
	}
	stat := t.theme[stats].(map[string]interface{})["memory"]

	if stat == nil {
		return memory
	}
	for name, i := range stat.(map[string]interface{}) {
		var interval int
		statTexts := make(map[string]entity.StatText)
		StatProgressBars := make(map[string]entity.StatProgressBar)
		StatRadialBars := make(map[string]entity.StatRadialBar)

		//fmt.Printf("[%s] - [%#v]\n", name, i)

		switch reflect.TypeOf(i).Kind() {
		case reflect.Int:
			if i == nil {
				interval = 1.0
			} else {
				interval = i.(int)
			}
		case reflect.Map:
			memory[name] = entity.Memory{
				Interval:         time.Duration(interval) * time.Second,
				StatTexts:        statTexts,
				StatProgressBars: StatProgressBars,
				StatRadialBars:   StatRadialBars,
			}
		}
	}
	return nil
}
func (t Theme) GetNetworkStats() map[string]entity.Network {
	network := make(map[string]entity.Network)

	if t.theme[stats] == nil {
		return network
	}
	stat := t.theme[stats].(map[string]interface{})["network"]

	if stat == nil {
		return network
	}
	for name, i := range stat.(map[string]interface{}) {
		var interval int
		statTexts := make(map[string]entity.StatText)
		StatProgressBars := make(map[string]entity.StatProgressBar)
		StatRadialBars := make(map[string]entity.StatRadialBar)

		//fmt.Printf("[%s] - [%#v]\n", name, i)

		switch reflect.TypeOf(i).Kind() {
		case reflect.Int:
			if i == nil {
				interval = 1.0
			} else {
				interval = i.(int)
			}
		case reflect.Map:
			network[name] = entity.Network{
				Interval:         time.Duration(interval) * time.Second,
				StatTexts:        statTexts,
				StatProgressBars: StatProgressBars,
				StatRadialBars:   StatRadialBars,
			}
		}
	}
	return nil
}
func (t Theme) GetDateTimeStats() map[string]entity.DateTime {
	date_time := make(map[string]entity.DateTime)

	if t.theme[stats] == nil {
		return date_time
	}

	return nil
}
