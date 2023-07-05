package entity

import (
	"golang.org/x/image/font"
	"image/color"
)

type Orientation int

const (
	PORTRAIT          Orientation = 0
	REVERSE_PORTRAIT  Orientation = 1
	LANDSCAPE         Orientation = 2
	REVERSE_LANDSCAPE Orientation = 3
)

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
	switch src {
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

type StaticImages struct {
	Height int
	Path   string
	Width  int
	X      int
	Y      int
}
type StaticTexts struct {
	Text            string
	Font            font.Face
	FontColor       color.Color
	BackgroundColor color.Color
	X               int
	Y               int
}
