package theme

import (
	"image"
	"image/color"
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

// Layout define a posição e as dimensões de um componente na tela.
type Layout struct {
	X      int `mapstructure:"X"`
	Y      int `mapstructure:"Y"`
	Width  int `mapstructure:"WIDTH"`
	Height int `mapstructure:"HEIGHT"`
	Index  int `mapstructure:"INDEX"`
}

// BackgroundStyle define o fundo de um componente.
type BackgroundStyle struct {
	BackgroundColor     color.Color
	BackgroundImagePath string `mapstructure:"BACKGROUND_IMAGE"`
	BackgroundImage     image.Image
}

// TextStyle define as propriedades de renderização de um texto.
type TextStyle struct {
	Font      font.Face
	FontColor color.Color
	Align     Alignment
}

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
	Display      *Display               `mapstructure:"display"`
	VideoPlay    *VideoPlay             `mapstructure:"video"`
	StaticImages map[string]StaticImage `mapstructure:"static_images"`
	StaticTexts  map[string]StaticText  `mapstructure:"static_texts"`
	Stats        *Stats                 `mapstructure:"STATS"`
}

type Display struct {
	Size        string      `mapstructure:"SIZE"`
	Orientation Orientation
	Width       int         `mapstructure:"WIDTH"`
	Height      int         `mapstructure:"HEIGHT"`
}

type VideoPlay struct {
	Layout
	Video           []byte
	Size            int64
	ForegroundImage image.Image
	Path            string `mapstructure:"PATH"`
	DevicePath      string
}

type StaticImage struct {
	Layout
	BackgroundImage image.Image
	Path            string `mapstructure:"PATH"`
}

type StaticText struct {
	Layout
	TextStyle
	BackgroundStyle
	Text string `mapstructure:"TEXT"`
}

type Stats struct {
	CPU     *CPU      `mapstructure:"CPU"`
	GPU     *GPU      `mapstructure:"GPU"`
	Memory  *Memory   `mapstructure:"MEMORY"`
	Disk    *Disk     `mapstructure:"DISK"`
	Net     *Network  `mapstructure:"NET"`
	Date    *DateTime `mapstructure:"DATE"`
	Weather *Weather  `mapstructure:"WEATHER"`
	Volume  *Volume   `mapstructure:"VOLUME"`
}

type Mesurement struct {
	Graph     *Graph     `mapstructure:"GRAPH"`
	Radial    *Radial    `mapstructure:"RADIAL"`
	Gauge     *Gauge     `mapstructure:"GAUGE"`
	StatusBar *StatusBar `mapstructure:"STATUS_BAR"`
	Chart     *Chart     `mapstructure:"CHART"`
	Text      *Text      `mapstructure:"TEXT"`
	Percent   *Text      `mapstructure:"PERCENT_TEXT"`
}

type Text struct {
	Layout
	BackgroundStyle
	TextStyle
	Show        bool   `mapstructure:"SHOW"`
	ShowUnit    bool   `mapstructure:"SHOW_UNIT"`
	Placeholder string `mapstructure:"PLACEHOLDER"`
	Format      Format
	Size        int
}

type Graph struct {
	Layout
	BackgroundStyle
	Show          bool   `mapstructure:"SHOW"`
	Direction     string `mapstructure:"DIRECTION"` // left (default), right, up, down
	MinValue      int    `mapstructure:"MIN_VALUE"`
	MaxValue      int    `mapstructure:"MAX_VALUE"`
	BarColor      color.Color
	EmptyColor    color.Color // color for unfilled track; nil = transparent
	GradientColor color.Color // vertical gradient top color (nil = no gradient)
	BarOutline    bool        `mapstructure:"BAR_OUTLINE"`
	BorderWidth   int         `mapstructure:"BORDER_WIDTH"`   // border thickness around full rect
	CornerRadius  int         `mapstructure:"CORNER_RADIUS"`  // rounded corner radius
	Steps         int         `mapstructure:"STEPS"`          // number of segments (0 = continuous)
	StepGap       int         `mapstructure:"STEP_GAP"`       // gap between segments in pixels
	BlockWidth    int         `mapstructure:"BLOCK_WIDTH"`    // block size in px (overrides Steps)
	RevertValue   bool        `mapstructure:"REVERT_VALUE"`   // invert ratio (show remaining)
}

type Radial struct {
	Layout
	BackgroundStyle
	TextStyle
	Show          bool `mapstructure:"SHOW"`
	Radius        int  `mapstructure:"RADIUS"`
	MinValue      int  `mapstructure:"MIN_VALUE"`
	MaxValue      int  `mapstructure:"MAX_VALUE"`
	AngleStart    int  `mapstructure:"ANGLE_START"`
	AngleEnd      int  `mapstructure:"ANGLE_END"`
	AngleSteps    int  `mapstructure:"ANGLE_STEPS"`
	AngleSep      int  `mapstructure:"ANGLE_SEP"`
	BlockAngle    int  `mapstructure:"BLOCK_ANGLE"`   // degrees per block (overrides AngleSteps)
	Clockwise     bool `mapstructure:"CLOCKWISE"`
	BarColor      color.Color
	EmptyColor    color.Color // color for unfilled track; nil = transparent
	GradientColor color.Color // vertical gradient top color (nil = no gradient)
	Round         bool        `mapstructure:"ROUND"`        // rounded arc endpoints
	Revert        bool        `mapstructure:"REVERT"`       // swap bar/empty colors
	RevertValue   bool        `mapstructure:"REVERT_VALUE"` // fill from arc end
	ShowText      bool        `mapstructure:"SHOW_TEXT"`
	ShowUnit      bool        `mapstructure:"SHOW_UNIT"`
}

// Gauge draws a needle/pointer at an angle (like a speedometer).
// Uses the same angle logic as Radial but renders a line from center.
type Gauge struct {
	Layout
	BackgroundStyle
	TextStyle
	Show        bool `mapstructure:"SHOW"`
	Radius      int  `mapstructure:"RADIUS"`
	NeedleWidth int  `mapstructure:"NEEDLE_WIDTH"`
	MinValue    int  `mapstructure:"MIN_VALUE"`
	MaxValue    int  `mapstructure:"MAX_VALUE"`
	AngleStart  int  `mapstructure:"ANGLE_START"`
	AngleEnd    int  `mapstructure:"ANGLE_END"`
	NeedleColor color.Color
	ShowText    bool `mapstructure:"SHOW_TEXT"`
	ShowUnit    bool `mapstructure:"SHOW_UNIT"`
}

// StatusBar draws a progress bar with an indicator (circle) at the current position.
type StatusBar struct {
	Layout
	BackgroundStyle
	Show            bool `mapstructure:"SHOW"`
	MinValue        int  `mapstructure:"MIN_VALUE"`
	MaxValue        int  `mapstructure:"MAX_VALUE"`
	BarColor        color.Color
	IndicatorColor  color.Color
	IndicatorRadius int `mapstructure:"INDICATOR_RADIUS"`
}

// Chart draws a time-series line/bar chart with a ring buffer of historical values.
type Chart struct {
	Layout
	BackgroundStyle
	Show        bool   `mapstructure:"SHOW"`
	Style       string `mapstructure:"STYLE"` // "bar" (default) or "line"
	MinValue    int    `mapstructure:"MIN_VALUE"`
	MaxValue    int    `mapstructure:"MAX_VALUE"`
	ColumnWidth int    `mapstructure:"COLUMN_WIDTH"`
	ColumnGap   int    `mapstructure:"COLUMN_GAP"`
	FillColor   color.Color
	LineColor   color.Color
	BorderWidth int `mapstructure:"BORDER_WIDTH"`
	MaxSamples  int `mapstructure:"MAX_SAMPLES"`
	// Runtime: ring buffer of values (not persisted)
	samples []float64
	pos     int
	full    bool
}

// AddSample adds a value to the chart ring buffer.
func (c *Chart) AddSample(value float64) {
	if c.MaxSamples <= 0 {
		// Calculate from dimensions
		step := c.ColumnWidth + c.ColumnGap
		if step <= 0 {
			step = 2
		}
		c.MaxSamples = c.Width / step
	}
	if c.samples == nil {
		c.samples = make([]float64, c.MaxSamples)
	}
	c.samples[c.pos] = value
	c.pos = (c.pos + 1) % c.MaxSamples
	if c.pos == 0 {
		c.full = true
	}
}

// GetSamples returns the samples in chronological order (oldest first).
func (c *Chart) GetSamples() []float64 {
	if c.samples == nil {
		return nil
	}
	if !c.full {
		return c.samples[:c.pos]
	}
	// Ring buffer: from pos to end, then from 0 to pos
	result := make([]float64, c.MaxSamples)
	copy(result, c.samples[c.pos:])
	copy(result[c.MaxSamples-c.pos:], c.samples[:c.pos])
	return result
}
