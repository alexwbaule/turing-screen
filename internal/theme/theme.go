package theme

import (
	"image"
	"strings"

	"golang.org/x/image/font"
)

type Orientation int
type Alignment int
type Format int
type FormatDateTime int

var (
	DefaultImagePath = "res/themes/"
	DefaultFontPath  = "res/fonts/"
)

const (
	PORTRAIT          Orientation    = 0
	REVERSE_PORTRAIT  Orientation    = 1
	LANDSCAPE         Orientation    = 2
	REVERSE_LANDSCAPE Orientation    = 3
	LEFT              Alignment      = 0
	CENTER            Alignment      = 1
	RIGHT             Alignment      = 2
	SHORT             Format         = 0
	MEDIUM            Format         = 1
	LONG              Format         = 2
	FULL              Format         = 3
	DATE              FormatDateTime = 0
	TIME              FormatDateTime = 1
)

func (a Alignment) String() string {
	switch a {
	case LEFT:
		return "LEFT"
	case CENTER:
		return "CENTER"
	case RIGHT:
		return "RIGHT"
	}
	return "LEFT"
}

func StringToFormat(src string) Format {
	switch strings.ToUpper(src) {
	case "SHORT":
		return SHORT
	case "MEDIUM":
		return MEDIUM
	case "LONG":
		return LONG
	case "FULL":
		return FULL
	}
	return SHORT
}

func (f Format) String(t FormatDateTime) string {
	switch f {
	case SHORT:
		if t == DATE {
			return "01/02/06"
		}
		return "15:04"
	case MEDIUM:
		if t == DATE {
			return "Jan 02, 2006"
		}
		return "15:04:05"
	case LONG:
		if t == DATE {
			return "January 02, 2006"
		}
		return "15:04:05 MST"
	case FULL:
		if t == DATE {
			return "Monday, January 02, 2006"
		}
		return "15:04:05 -07:00:00"
	}
	if t == DATE {
		return "01/02/06"
	}
	return "15:04"
}

func StringToAlignment(src string) Alignment {
	switch strings.ToUpper(src) {
	case "LEFT":
		return LEFT
	case "CENTER":
		return CENTER
	case "RIGHT":
		return RIGHT
	}
	return LEFT
}

func (o Orientation) String() string {
	switch o {
	case PORTRAIT:
		return "PORTRAIT"
	case REVERSE_PORTRAIT:
		return "REVERSE_PORTRAIT"
	case LANDSCAPE:
		return "LANDSCAPE"
	case REVERSE_LANDSCAPE:
		return "REVERSE_LANDSCAPE"
	}
	return "LANDSCAPE"
}

func StringToOrientation(src string, reverse bool) Orientation {
	switch strings.ToUpper(src) {
	case "PORTRAIT":
		if reverse {
			return REVERSE_PORTRAIT
		}
		return PORTRAIT
	case "REVERSE_PORTRAIT":
		return REVERSE_PORTRAIT
	case "LANDSCAPE":
		if reverse {
			return REVERSE_LANDSCAPE
		}
		return LANDSCAPE
	case "REVERSE_LANDSCAPE":
		return REVERSE_LANDSCAPE
	}
	return LANDSCAPE
}

type Theme struct {
	Display      *Display               `yaml:"display"`
	StaticImages map[string]StaticImage `yaml:"static_images,omitempty"`
	VideoPlay    *DinamicImage          `yaml:"video,omitempty"`
	StaticText   map[string]Text        `yaml:"static_texts,omitempty"`
	Stats        *Stats                 `yaml:"STATS"`
}

type Display struct {
	Size        string `yaml:"SIZE"`
	Orientation string `yaml:"ORIENTATION"`
	RGBLed      string `yaml:"RGB_LED,omitempty"`
}

type DinamicImage struct {
	Path       string `yaml:"PATH"`
	Height     int    `yaml:"HEIGHT"`
	Width      int    `yaml:"WIDTH"`
	X          int    `yaml:"X"`
	Y          int    `yaml:"Y"`
	Background string `yaml:"BACKGROUND,omitempty"`
}

type StaticImage struct {
	Path   string `yaml:"PATH"`
	Height int    `yaml:"HEIGHT"`
	Width  int    `yaml:"WIDTH"`
	X      int    `yaml:"X"`
	Y      int    `yaml:"Y"`
}

type Stats struct {
	CPU     *CPU      `yaml:"CPU"`
	GPU     *GPU      `yaml:"GPU"`
	Memory  *Memory   `yaml:"MEMORY"`
	Disk    *Disk     `yaml:"DISK"`
	Net     *Network  `yaml:"NET"`
	Date    *DateTime `yaml:"DATE"`
	Weather *Weather  `yaml:"WEATHER,omitempty"`
	Volume  *Volume   `yaml:"VOLUME,omitempty"`
}

type Weather struct {
	Temperature *Measurement `yaml:"TEMPERATURE,omitempty"`
	Condition   *Text        `yaml:"CONDITION,omitempty"`
}

type Volume struct {
	Text *Text `yaml:"TEXT,omitempty"`
}

type Measurement struct {
	Show        bool       `yaml:"SHOW"`
	Graph       *Graph     `yaml:"GRAPH,omitempty"`
	Radial      *Radial    `yaml:"RADIAL,omitempty"`
	Gauge       *Gauge     `yaml:"GAUGE,omitempty"`
	StatusBar   *StatusBar `yaml:"STATUS_BAR,omitempty"`
	Chart       *Chart     `yaml:"CHART,omitempty"`
	Text        *Text      `yaml:"TEXT,omitempty"`
	PercentText *Text      `yaml:"PERCENT_TEXT,omitempty"`
}

type Text struct {
	Text            string      `yaml:"TEXT,omitempty"`
	Placeholder     string      `yaml:"PLACEHOLDER,omitempty"`
	Show            bool        `yaml:"SHOW"`
	ShowUnit        bool        `yaml:"SHOW_UNIT"`
	Format          string      `yaml:"FORMAT,omitempty"`
	Index           int         `yaml:"INDEX,omitempty"`
	X               int         `yaml:"X"`
	Y               int         `yaml:"Y"`
	Font            string      `yaml:"FONT,omitempty"`
	FontSize        int         `yaml:"FONT_SIZE,omitempty"`
	FontColor       string      `yaml:"FONT_COLOR,omitempty"`
	BackgroundColor string      `yaml:"BACKGROUND_COLOR,omitempty"`
	Align           string      `yaml:"ALIGN,omitempty"`
	LoadedFont      font.Face   `yaml:"-"`
	LoadedImage     image.Image `yaml:"-"`
}

type Graph struct {
	Show            bool        `yaml:"SHOW"`
	Index           int         `yaml:"INDEX,omitempty"`
	Direction       string      `yaml:"DIRECTION,omitempty"`
	X               int         `yaml:"X"`
	Y               int         `yaml:"Y"`
	Width           int         `yaml:"WIDTH"`
	Height          int         `yaml:"HEIGHT"`
	MinValue        int         `yaml:"MIN_VALUE"`
	MaxValue        int         `yaml:"MAX_VALUE"`
	BarColor        string      `yaml:"BAR_COLOR"`
	BackgroundColor string      `yaml:"BACKGROUND_COLOR,omitempty"`
	BarOutline      bool        `yaml:"BAR_OUTLINE"`
	Steps           int         `yaml:"STEPS,omitempty"`
	StepGap         int         `yaml:"STEP_GAP,omitempty"`
	LoadedImage     image.Image `yaml:"-"`
}

type Radial struct {
	Show            bool        `yaml:"SHOW"`
	Index           int         `yaml:"INDEX,omitempty"`
	X               int         `yaml:"X"`
	Y               int         `yaml:"Y"`
	Radius          int         `yaml:"RADIUS"`
	Width           int         `yaml:"WIDTH"`
	MinValue        int         `yaml:"MIN_VALUE"`
	MaxValue        int         `yaml:"MAX_VALUE"`
	AngleStart      int         `yaml:"ANGLE_START"`
	AngleEnd        int         `yaml:"ANGLE_END"`
	AngleSteps      int         `yaml:"ANGLE_STEPS"`
	AngleSep        int         `yaml:"ANGLE_SEP"`
	Clockwise       bool        `yaml:"CLOCKWISE"`
	BarColor        string      `yaml:"BAR_COLOR"`
	ShowText        bool        `yaml:"SHOW_TEXT"`
	ShowUnit        bool        `yaml:"SHOW_UNIT"`
	Font            string      `yaml:"FONT,omitempty"`
	FontColor       string      `yaml:"FONT_COLOR,omitempty"`
	BackgroundColor string      `yaml:"BACKGROUND_COLOR,omitempty"`
	LoadedFont      font.Face   `yaml:"-"`
	LoadedImage     image.Image `yaml:"-"`
}

type Chart struct {
	Show        bool        `yaml:"SHOW"`
	Style       string      `yaml:"STYLE,omitempty"`
	Index       int         `yaml:"INDEX,omitempty"`
	X           int         `yaml:"X"`
	Y           int         `yaml:"Y"`
	Width       int         `yaml:"WIDTH"`
	Height      int         `yaml:"HEIGHT"`
	MinValue    int         `yaml:"MIN_VALUE"`
	MaxValue    int         `yaml:"MAX_VALUE"`
	ColumnWidth int         `yaml:"COLUMN_WIDTH"`
	ColumnGap   int         `yaml:"COLUMN_GAP"`
	FillColor   string      `yaml:"FILL_COLOR"`
	LineColor   string      `yaml:"LINE_COLOR"`
	BorderWidth int         `yaml:"BORDER_WIDTH"`
	MaxSamples  int         `yaml:"MAX_SAMPLES,omitempty"`
	LoadedImage image.Image `yaml:"-"`
}

type Gauge struct {
	Show            bool        `yaml:"SHOW"`
	Index           int         `yaml:"INDEX,omitempty"`
	X               int         `yaml:"X"`
	Y               int         `yaml:"Y"`
	Radius          int         `yaml:"RADIUS"`
	NeedleWidth     int         `yaml:"NEEDLE_WIDTH"`
	MinValue        int         `yaml:"MIN_VALUE"`
	MaxValue        int         `yaml:"MAX_VALUE"`
	AngleStart      int         `yaml:"ANGLE_START"`
	AngleEnd        int         `yaml:"ANGLE_END"`
	NeedleColor     string      `yaml:"NEEDLE_COLOR"`
	ShowText        bool        `yaml:"SHOW_TEXT"`
	ShowUnit        bool        `yaml:"SHOW_UNIT"`
	Font            string      `yaml:"FONT,omitempty"`
	FontColor       string      `yaml:"FONT_COLOR,omitempty"`
	BackgroundColor string      `yaml:"BACKGROUND_COLOR,omitempty"`
	LoadedFont      font.Face   `yaml:"-"`
	LoadedImage     image.Image `yaml:"-"`
}

type StatusBar struct {
	Show            bool        `yaml:"SHOW"`
	Index           int         `yaml:"INDEX,omitempty"`
	X               int         `yaml:"X"`
	Y               int         `yaml:"Y"`
	Width           int         `yaml:"WIDTH"`
	Height          int         `yaml:"HEIGHT"`
	MinValue        int         `yaml:"MIN_VALUE"`
	MaxValue        int         `yaml:"MAX_VALUE"`
	BarColor        string      `yaml:"BAR_COLOR"`
	IndicatorColor  string      `yaml:"INDICATOR_COLOR"`
	IndicatorRadius int         `yaml:"INDICATOR_RADIUS"`
	BackgroundColor string      `yaml:"BACKGROUND_COLOR,omitempty"`
	LoadedImage     image.Image `yaml:"-"`
}
