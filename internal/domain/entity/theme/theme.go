package theme

import (
	"golang.org/x/image/font"
	"image"
	"image/color"
	"strings"
	"time"
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
		//	//DATE: short (2/20/23) / medium (Feb 20, 2023) / long (February 20, 2023) / full (Monday, February 20, 2023)
		//	//TIME: short (6:48 PM) / medium (6:48:53 PM) / long (6:48:53 PM UTC) / full (6:48:53 PM Coordinated Universal Time)
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
	CPU    *CPU      `mapstructure:"CPU"`
	GPU    *GPU      `mapstructure:"GPU"`
	Memory *Memory   `mapstructure:"MEMORY"`
	Disk   *Disk     `mapstructure:"DISK"`
	Net    *Network  `mapstructure:"NET"`
	Date   *DateTime `mapstructure:"DATE"`
}

type Mesurement struct {
	Interval time.Duration `mapstructure:"INTERVAL"`
	Graph    *Graph        `mapstructure:"GRAPH"`
	Radial   *Radial       `mapstructure:"RADIAL"`
	Text     *Text         `mapstructure:"TEXT"`
	Percent  *Text         `mapstructure:"PERCENT_TEXT"`
}

type Text struct {
	Show                bool `mapstructure:"SHOW"`
	ShowUnit            bool `mapstructure:"SHOW_UNIT"`
	Format              Format
	X                   int `mapstructure:"X"`
	Y                   int `mapstructure:"Y"`
	Font                font.Face
	FontColor           color.Color
	BackgroundColor     color.Color
	BackgroundImagePath string `mapstructure:"BACKGROUND_IMAGE"`
	BackgroundImage     image.Image
	Align               Alignment
	Size                int
}

type Graph struct {
	Show                bool `mapstructure:"SHOW"`
	X                   int  `mapstructure:"X"`
	Y                   int  `mapstructure:"Y"`
	Width               int  `mapstructure:"WIDTH"`
	Height              int  `mapstructure:"HEIGHT"`
	MinValue            int  `mapstructure:"MIN_VALUE"`
	MaxValue            int  `mapstructure:"MAX_VALUE"`
	BarColor            color.Color
	BarOutline          bool   `mapstructure:"BAR_OUTLINE"`
	BackgroundImagePath string `mapstructure:"BACKGROUND_IMAGE"`
	BackgroundImage     image.Image
}

type Radial struct {
	Show                bool `mapstructure:"SHOW"`
	X                   int  `mapstructure:"X"`
	Y                   int  `mapstructure:"Y"`
	Radius              int  `mapstructure:"RADIUS"`
	Width               int  `mapstructure:"WIDTH"`
	MinValue            int  `mapstructure:"MIN_VALUE"`
	MaxValue            int  `mapstructure:"MAX_VALUE"`
	AngleStart          int  `mapstructure:"ANGLE_START"`
	AngleEnd            int  `mapstructure:"ANGLE_END"`
	AngleSteps          int  `mapstructure:"ANGLE_STEPS"`
	AngleStep           int  `mapstructure:"ANGLE_SEP"`
	Clockwise           bool `mapstructure:"CLOCKWISE"`
	BarColor            color.Color
	ShowText            bool `mapstructure:"SHOW_TEXT"`
	ShowUnit            bool `mapstructure:"SHOW_UNIT"`
	Font                font.Face
	FontColor           color.Color
	BackgroundColor     color.Color
	BackgroundImagePath string `mapstructure:"BACKGROUND_IMAGE"`
	BackgroundImage     image.Image
}
