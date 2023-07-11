package theme

import (
	"golang.org/x/image/font"
	"image/color"
	"strings"
	"time"
)

type Orientation int
type Alignment int

var (
	DefaultImagePath = "res/themes/"
	DefaultFontPath  = "res/fonts/"
)

const (
	PORTRAIT          Orientation = 0
	REVERSE_PORTRAIT  Orientation = 1
	LANDSCAPE         Orientation = 2
	REVERSE_LANDSCAPE Orientation = 3
	LEFT              Alignment   = 0
	CENTER            Alignment   = 1
	RIGHT             Alignment   = 2
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

func StringToOrientation(src string) Orientation {
	switch strings.ToUpper(src) {
	case "PORTRAIT":
		return PORTRAIT
	case "REVERSE_PORTRAIT":
		return REVERSE_PORTRAIT
	case "LANDSCAPE":
		return LANDSCAPE
	case "REVERSE_LANDSCAPE":
		return REVERSE_LANDSCAPE
	}
	return LANDSCAPE
}

type Theme struct {
	Display      *Display               `mapstructure:"display"`
	StaticImages map[string]StaticImage `mapstructure:"static_images"`
	StaticTexts  map[string]StaticText  `mapstructure:"static_texts"`
	Stats        *Stats                 `mapstructure:"STATS"`
}

type Display struct {
	Size        string `mapstructure:"SIZE"`
	Orientation Orientation
}

type StaticImage struct {
	Path   string `mapstructure:"PATH"`
	Height int    `mapstructure:"HEIGHT"`
	Width  int    `mapstructure:"WIDTH"`
	X      int    `mapstructure:"X"`
	Y      int    `mapstructure:"Y"`
}

type StaticText struct {
	Text            string `mapstructure:"TEXT"`
	X               int    `mapstructure:"X"`
	Y               int    `mapstructure:"Y"`
	Font            font.Face
	FontColor       color.Color
	BackgroundColor color.Color
}

type Stats struct {
	CPU    *CPU     `mapstructure:"CPU"`
	GPU    *GPU     `mapstructure:"GPU"`
	Memory *Memory  `mapstructure:"MEMORY"`
	Disk   *Disk    `mapstructure:"DISK"`
	Net    *Network `mapstructure:"NET"`
	Date   DateTime `mapstructure:"DATE"`
}

type Mesurement struct {
	Interval time.Duration `mapstructure:"INTERVAL"`
	Graph    *Graph        `mapstructure:"GRAPH"`
	Radial   *Radial       `mapstructure:"RADIAL"`
	Text     *Text         `mapstructure:"TEXT"`
	Percent  *Text         `mapstructure:"PERCENT_TEXT"`
}

type Text struct {
	Show            bool `mapstructure:"SHOW"`
	ShowUnit        bool `mapstructure:"SHOW_UNIT"`
	X               int  `mapstructure:"X"`
	Y               int  `mapstructure:"Y"`
	Font            font.Face
	FontColor       color.Color
	BackgroundColor color.Color
	BackgroundImage string `mapstructure:"BACKGROUND_IMAGE"`
	Align           Alignment
	Size            int
}

type Graph struct {
	Show            bool `mapstructure:"SHOW"`
	X               int  `mapstructure:"X"`
	Y               int  `mapstructure:"Y"`
	Width           int  `mapstructure:"WIDTH"`
	Height          int  `mapstructure:"HEIGHT"`
	MinValue        int  `mapstructure:"MIN_VALUE"`
	MaxValue        int  `mapstructure:"MAX_VALUE"`
	BarColor        color.Color
	BarOutline      bool   `mapstructure:"BAR_OUTLINE"`
	BackgroundImage string `mapstructure:"BACKGROUND_IMAGE"`
}

type Radial struct {
	Show            bool `mapstructure:"SHOW"`
	X               int  `mapstructure:"X"`
	Y               int  `mapstructure:"Y"`
	Radius          int  `mapstructure:"RADIUS"`
	Width           int  `mapstructure:"WIDTH"`
	MinValue        int  `mapstructure:"MIN_VALUE"`
	MaxValue        int  `mapstructure:"MAX_VALUE"`
	AngleStart      int  `mapstructure:"ANGLE_START"`
	AngleEnd        int  `mapstructure:"ANGLE_END"`
	AngleSteps      int  `mapstructure:"ANGLE_STEPS"`
	AngleStep       int  `mapstructure:"ANGLE_SEP"`
	Clockwise       bool `mapstructure:"CLOCKWISE"`
	BarColor        color.Color
	ShowText        bool `mapstructure:"SHOW_TEXT"`
	ShowUnit        bool `mapstructure:"SHOW_UNIT"`
	Font            font.Face
	FontColor       color.Color
	BackgroundColor color.Color
	BackgroundImage string `mapstructure:"BACKGROUND_IMAGE"`
}
