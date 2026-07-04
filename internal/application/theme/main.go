package theme

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/config"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/utils"
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

	return &Theme{
		theme:    cconfig,
		path:     fmt.Sprintf("res/themes/%s/", file),
		fontPath: "res/fonts/",
		log:      l,
	}, nil
}

// getInt / getBool / getString read an optional key from a decoded theme map
// without panicking when the key is absent. A theme may legitimately omit
// optional fields (e.g. BAR_OUTLINE), and Go's zero value is the sensible
// default for every one of them — so returning 0/false/"" on a miss matches
// the daemon's expected behaviour instead of crashing on a nil interface.
func getInt(data map[string]interface{}, key string) int {
	if val, ok := data[key].(int); ok {
		return val
	}
	return 0
}

func getBool(data map[string]interface{}, key string) bool {
	if val, ok := data[key].(bool); ok {
		return val
	}
	return false
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
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
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.Gauge{}) || t == reflect.TypeOf(&theme.Gauge{})) {
			return translateGauge(file, data.(map[string]interface{}))
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.StatusBar{}) || t == reflect.TypeOf(&theme.StatusBar{})) {
			return translateStatusBar(file, data.(map[string]interface{}))
		}
		if f.Kind() == reflect.Map && (t == reflect.TypeOf(theme.Chart{}) || t == reflect.TypeOf(&theme.Chart{})) {
			return translateChart(file, data.(map[string]interface{}))
		}
		return data, nil
	}
}

// parseOptionalColor parses a "r, g, b" or "#RRGGBB" string only when non-empty
// and only when the resulting color has alpha > 0. Returns nil otherwise, which
// lets callers distinguish "no color" from "transparent" and avoids enabling
// gradient mode when GRADIENT_COLOR is '' in the YAML.
func parseOptionalColor(data map[string]interface{}, key string) color.Color {
	val, ok := data[key].(string)
	if !ok || val == "" {
		return nil
	}
	c := utils.ConvertToColor(val, color.Transparent)
	_, _, _, a := c.RGBA()
	if a == 0 {
		return nil
	}
	return c
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

	index := 0
	if val, ok := data["index"].(int); ok {
		index = val
	}

	return theme.Layout{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		Index:  index,
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

func parseTextStyle(data map[string]interface{}, defaultAlign theme.Alignment) theme.TextStyle {
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

	if val, ok := data["align"].(string); ok && val != "" {
		align = theme.StringToAlignment(val)
	} else {
		align = defaultAlign
	}

	return theme.TextStyle{
		Font:      fface,
		FontColor: fColor,
		Align:     align,
	}
}


func translateDuration(data interface{}) (interface{}, error) {
	const minInterval = 100 * time.Millisecond

	switch val := data.(type) {
	case string:
		d, err := time.ParseDuration(val)
		if err != nil {
			return minInterval, nil
		}
		if d < minInterval {
			return minInterval, nil
		}
		return d, nil
	case int:
		d := time.Duration(val) * time.Second
		if d < minInterval {
			return minInterval, nil
		}
		return d, nil
	}
	return 1 * time.Second, nil
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

	gradientColor := parseOptionalColor(data, "gradient_color")

	show := false
	if val, ok := data["show"].(bool); ok {
		show = val
	}

	steps := 0
	if val, ok := data["steps"].(int); ok {
		steps = val
	}
	stepGap := 0
	if val, ok := data["step_gap"].(int); ok {
		stepGap = val
	}
	blockWidth := 0
	if val, ok := data["block_width"].(int); ok {
		blockWidth = val
	}
	cornerRadius := 0
	if val, ok := data["corner_radius"].(int); ok {
		cornerRadius = val
	}
	borderWidth := 0
	if val, ok := data["border_width"].(int); ok {
		borderWidth = val
	}
	revertValue := false
	if val, ok := data["revert_value"].(bool); ok {
		revertValue = val
	}
	direction := ""
	if val, ok := data["direction"].(string); ok {
		direction = val
	}

	v := theme.Graph{
		Layout:          layout,
		BackgroundStyle: bgStyle,
		Show:            show,
		Direction:       direction,
		MinValue:        getInt(data, "min_value"),
		MaxValue:        getInt(data, "max_value"),
		BarColor:        barColor,
		GradientColor:   gradientColor,
		BarOutline:      getBool(data, "bar_outline"),
		Steps:           steps,
		StepGap:         stepGap,
		BlockWidth:      blockWidth,
		CornerRadius:    cornerRadius,
		BorderWidth:     borderWidth,
		RevertValue:     revertValue,
	}
	return v, nil
}

func translateRadial(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	bgStyle, err := parseBackgroundStyle(data, file)
	if err != nil {
		return nil, err
	}
	textStyle := parseTextStyle(data, theme.LEFT)

	var barColor color.Color
	if val, ok := data["bar_color"].(string); ok {
		barColor = utils.ConvertToColor(val, color.Transparent)
	} else {
		barColor = color.Transparent
	}

	gradientColor := parseOptionalColor(data, "gradient_color")

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

	round := false
	if val, ok := data["round"].(bool); ok {
		round = val
	}
	revert := false
	if val, ok := data["revert"].(bool); ok {
		revert = val
	}
	revertValue := false
	if val, ok := data["revert_value"].(bool); ok {
		revertValue = val
	}
	blockAngle := 0
	if val, ok := data["block_angle"].(int); ok {
		blockAngle = val
	}
	// Default true (matches the Python editor's CLOCKWISE dataclass default and
	// the direction every existing theme already relies on) — unlike the other
	// bools above, false is not a safe zero-value default here since it now
	// flips the arc to the complementary span instead of just changing intent.
	clockwise := true
	if val, ok := data["clockwise"].(bool); ok {
		clockwise = val
	}

	v := theme.Radial{
		Layout:          layout,
		BackgroundStyle: bgStyle,
		TextStyle:       textStyle,
		Show:            show,
		Radius:          getInt(data, "radius"),
		MinValue:        getInt(data, "min_value"),
		MaxValue:        getInt(data, "max_value"),
		AngleStart:      getInt(data, "angle_start"),
		AngleEnd:        getInt(data, "angle_end"),
		AngleSteps:      getInt(data, "angle_steps"),
		AngleSep:        getInt(data, "angle_sep"),
		BlockAngle:      blockAngle,
		Clockwise:       clockwise,
		BarColor:        barColor,
		GradientColor:   gradientColor,
		Round:           round,
		Revert:          revert,
		RevertValue:     revertValue,
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
	var w, h int
	if val, ok := data["width"].(int); ok {
		w = val
	}
	if val, ok := data["height"].(int); ok {
		h = val
	}
	v := theme.Display{
		Size:        size,
		Orientation: theme.StringToOrientation(orientation, reverse),
		Width:       w,
		Height:      h,
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
		bVideo, err := utils.LoadH264(bgImagePath)
		if err != nil {
			return theme.BackgroundStyle{}, err
		}
		return theme.VideoPlay{
			Layout:          layout,
			Video:           bVideo,
			Size:            int64(len(bVideo)),
			ForegroundImage: image.Transparent,
			Path:            bgImagePath,    // full path relative to project root (used by Rev-C initializer)
			DevicePath:      "/root/video/" + val,
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
	textStyle := parseTextStyle(data, theme.LEFT)

	v := theme.StaticText{
		Layout:          layout,
		TextStyle:       textStyle,
		BackgroundStyle: bgStyle,
		Text:            getString(data, "text"),
	}
	return v, nil
}

func translateText(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	bgStyle, err := parseBackgroundStyle(data, file)
	if err != nil {
		return nil, err
	}
	textStyle := parseTextStyle(data, theme.LEFT)

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

	size := 0
	if val, ok := data["size"].(int); ok {
		size = val
	}

	v := theme.Text{
		Layout:          layout,
		BackgroundStyle: bgStyle,
		TextStyle:       textStyle,
		Show:            show,
		ShowUnit:        showUnit,
		Format:          format,
		Size:            size,
	}
	return v, nil
}

func translateGauge(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	bgStyle, err := parseBackgroundStyle(data, file)
	if err != nil {
		return nil, err
	}
	textStyle := parseTextStyle(data, theme.LEFT)

	var needleColor color.Color
	if val, ok := data["needle_color"].(string); ok {
		needleColor = utils.ConvertToColor(val, color.White)
	} else {
		needleColor = color.White
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
	needleWidth := 2
	if val, ok := data["needle_width"].(int); ok {
		needleWidth = val
	}

	v := theme.Gauge{
		Layout:          layout,
		BackgroundStyle: bgStyle,
		TextStyle:       textStyle,
		Show:            show,
		Radius:          getInt(data, "radius"),
		NeedleWidth:     needleWidth,
		MinValue:        getInt(data, "min_value"),
		MaxValue:        getInt(data, "max_value"),
		AngleStart:      getInt(data, "angle_start"),
		AngleEnd:        getInt(data, "angle_end"),
		NeedleColor:     needleColor,
		ShowText:        showText,
		ShowUnit:        showUnit,
	}
	return v, nil
}

func translateStatusBar(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	bgStyle, err := parseBackgroundStyle(data, file)
	if err != nil {
		return nil, err
	}

	var barColor color.Color
	if val, ok := data["bar_color"].(string); ok {
		barColor = utils.ConvertToColor(val, color.White)
	} else {
		barColor = color.White
	}

	var indicatorColor color.Color
	if val, ok := data["indicator_color"].(string); ok {
		indicatorColor = utils.ConvertToColor(val, color.White)
	} else {
		indicatorColor = color.White
	}

	show := false
	if val, ok := data["show"].(bool); ok {
		show = val
	}

	indicatorRadius := 5
	if val, ok := data["indicator_radius"].(int); ok {
		indicatorRadius = val
	}

	v := theme.StatusBar{
		Layout:          layout,
		BackgroundStyle: bgStyle,
		Show:            show,
		MinValue:        getInt(data, "min_value"),
		MaxValue:        getInt(data, "max_value"),
		BarColor:        barColor,
		IndicatorColor:  indicatorColor,
		IndicatorRadius: indicatorRadius,
	}
	return v, nil
}

func translateChart(file string, data map[string]interface{}) (interface{}, error) {
	layout := parseLayout(data)
	bgStyle, err := parseBackgroundStyle(data, file)
	if err != nil {
		return nil, err
	}

	var fillColor color.Color
	if val, ok := data["fill_color"].(string); ok {
		fillColor = utils.ConvertToColor(val, color.White)
	} else {
		fillColor = color.White
	}

	var lineColor color.Color
	if val, ok := data["line_color"].(string); ok {
		lineColor = utils.ConvertToColor(val, color.White)
	} else {
		lineColor = color.White
	}

	show := false
	if val, ok := data["show"].(bool); ok {
		show = val
	}

	columnWidth := 3
	if val, ok := data["column_width"].(int); ok {
		columnWidth = val
	}
	columnGap := 1
	if val, ok := data["column_gap"].(int); ok {
		columnGap = val
	}
	borderWidth := 0
	if val, ok := data["border_width"].(int); ok {
		borderWidth = val
	}
	maxSamples := 0
	if val, ok := data["max_samples"].(int); ok {
		maxSamples = val
	}

	style := "bar"
	if val, ok := data["style"].(string); ok {
		style = val
	}

	v := theme.Chart{
		Layout:          layout,
		BackgroundStyle: bgStyle,
		Show:            show,
		Style:           style,
		MinValue:        getInt(data, "min_value"),
		MaxValue:        getInt(data, "max_value"),
		ColumnWidth:     columnWidth,
		ColumnGap:       columnGap,
		FillColor:       fillColor,
		LineColor:       lineColor,
		BorderWidth:     borderWidth,
		MaxSamples:      maxSamples,
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

func (t *Theme) GetVideoPlay() *theme.VideoPlay {
	return t.theme.VideoPlay
}
