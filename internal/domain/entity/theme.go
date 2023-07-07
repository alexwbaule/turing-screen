package entity

import (
	"golang.org/x/image/font"
	"image/color"
	"strings"
)

type Orientation int
type Alignment int

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

type StaticImage struct {
	Path   string
	Height int
	Width  int
	X      int
	Y      int
}
type StaticText struct {
	Text            string
	Font            font.Face
	FontColor       color.Color
	BackgroundColor color.Color
	X               int
	Y               int
}

type StatText struct {
	Show            bool
	ShowUnit        bool
	X               int
	Y               int
	Align           Alignment
	Padding         int
	Font            font.Face
	FontColor       color.Color
	BackgroundColor color.Color
}

type StatProgressBar struct {
	Show            bool
	ShowUnit        bool
	X               int
	Y               int
	Width           int
	Height          int
	MinValue        int
	MaxValue        int
	Color           color.Color
	Outline         bool
	BackgroundColor color.Color
}

type StatRadialBar struct {
	Show            bool
	ShowUnit        bool
	X               int
	Y               int
	Radius          int
	Width           int
	MinValue        int
	MaxValue        int
	AngleStart      int
	AngleEnd        int
	AngleSteps      int
	AngleStep       int
	Color           color.Color
	BackgroundColor color.Color
	ShowText        bool
	Font            font.Face
	FontColor       color.Color
}
