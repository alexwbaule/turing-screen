package utils

import (
	"fmt"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strconv"
	"strings"
)

var DefaultFont = DefaultFontFace()

type Formatter interface {
	Hertz(s float64) string
	Bitsf(s float64) string
	Bytesf(s float64) string
	IBytesf(s float64) string
	Bits(s uint64) string
	Bytes(s uint64) string
	IBytes(s uint64) string
}

func CountStr(s string) int {
	return len([]rune(s))
}

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

func Radians(degrees int) float64 {
	return float64(degrees) * (math.Pi / 180.0)
}

func Degrees(radian float64) int {
	return int(radian * (180.0 / math.Pi))
}

func CreateImage(width int, height int, background color.Color) *image.RGBA {
	rect := image.Rect(0, 0, width, height)
	img := image.NewRGBA(rect)
	draw.Draw(img, img.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
	return img
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
		DPI:     72,
		Hinting: font.HintingFull,
	})
	return face
}

func DefaultFontFace() font.Face {
	fontBytes, err := os.ReadFile("res/fonts/jetbrains-mono/JetBrainsMono-Bold.ttf")
	if err != nil {
		return basicfont.Face7x13
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		return basicfont.Face7x13
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size:    23,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	return face
}

func Hertz(s float64) string {
	sizes := []string{"MHz", "GHz", "THz", "PHz", "EHz"}
	return humanateHertz(s, 1000, sizes)
}

func Bitsf(s float64) string {
	sizes := []string{"b", "kb", "Mb", "Gb", "Tb", "Pb", "Eb"}
	return humanateFloatBytes(s, 1000, sizes)
}

func Bytesf(s float64) string {
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	return humanateFloatBytes(s, 1000, sizes)
}

func IBytesf(s float64) string {
	sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	return humanateFloatBytes(s, 1024, sizes)
}

func Bits(s uint64) string {
	sizes := []string{"b", "kb", "Mb", "Gb", "Tb", "Pb", "Eb"}
	return humanateBytes(s, 1000, sizes)
}

func Bytes(s uint64) string {
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(s, 1000, sizes)
}

func IBytes(s uint64) string {
	sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	return humanateBytes(s, 1024, sizes)
}

func humanateHertz(s float64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%5d%s", s, sizes[0])
	}
	e := math.Floor(logn(s, base))
	suffix := sizes[int(e)]
	val := math.Floor(s/math.Pow(base, e)*10) / 10
	f := "%5.2f%s"
	if CountStr(suffix) == 3 {
		f = "%4.2f%s"
		if val < 10 {
			f = "%3.2f%s"
		}
	}
	if val < 10 {
		f = "%4.2f%s"
	}
	return fmt.Sprintf(f, val, suffix)
}

func humanateBytes(s uint64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%5d%s", s, sizes[0])
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%5.f%s"
	if CountStr(suffix) == 2 {
		f = "%4.f%s"
		if val < 10 {
			f = "%3.1f%s"
		}
	}
	if val < 10 {
		f = "%4.1f%s"
	}
	return fmt.Sprintf(f, val, suffix)
}

func humanateFloatBytes(s float64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%5d%s", s, sizes[0])
	}
	e := math.Floor(logn(s, base))
	suffix := sizes[int(e)]
	val := math.Floor(s/math.Pow(base, e)*10+0.5) / 10
	f := "%5.f%s"
	if CountStr(suffix) == 2 {
		f = "%4.f%s"
		if val < 10 {
			f = "%3.1f%s"
		}
	}
	if val < 10 {
		f = "%4.1f%s"
	}
	return fmt.Sprintf(f, val, suffix)
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}
