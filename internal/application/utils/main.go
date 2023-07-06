package utils

import (
	"fmt"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"
)

var DefaultFont = DefaultFontFace()

func IsInteger(val float64) bool {
	return val == float64(int(val))
}

func BZero(s int, b byte) []byte {
	tmp := make([]byte, s)
	for i := 0; i < s; i++ {
		tmp[i] = b
	}
	return tmp
}

func PadBegin(bb []byte, size int) []byte {
	l := len(bb)
	if l == size {
		return bb
	}
	if l > size {
		return bb
	}
	tmp := make([]byte, size)
	copy(tmp[size-l:], bb)
	return tmp
}

func LoadImage(path string) (image.Image, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %s", path, err)
	}
	defer reader.Close()
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %s", path, err)
	}
	return img, nil
}

func ConvertToColor(s string, dft color.Color) color.Color {

	rgba := strings.SplitN(s, ", ", 4)

	if len(rgba) == 3 {
		rgba = append(rgba, "255")
	}
	r, err := strconv.Atoi(rgba[0])
	if err != nil {
		return dft
	}
	g, err := strconv.Atoi(rgba[1])
	if err != nil {
		return dft
	}
	b, err := strconv.Atoi(rgba[2])
	if err != nil {
		return dft
	}
	a, err := strconv.Atoi(rgba[3])
	if err != nil {
		return dft
	}
	return color.RGBA{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: uint8(a),
	}
}

func LoadFontFace(path string, points float64) font.Face {
	fontBytes, err := os.ReadFile(path)
	if err != nil {
		return DefaultFontFace()
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		return DefaultFontFace()
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size:    points,
		Hinting: font.HintingFull,
	})
	return face
}

func DefaultFontFace() font.Face {
	fontBytes, err := os.ReadFile("res/fonts/roboto-mono/RobotoMono-Regular.ttf")
	if err != nil {
		return basicfont.Face7x13
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		return basicfont.Face7x13
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size:    20,
		Hinting: font.HintingFull,
	})
	return face
}
