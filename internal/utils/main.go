package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

var DefaultFont = DefaultFontFace()

func CountStr(s string) int {
	return len([]rune(s))
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

func LoadVideo(path string) ([]byte, int64, error) {
	if path == "" {
		return nil, 0, fmt.Errorf("path is empty")
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to stat %s: %s", path, err)
	}
	reader, err := os.Open(path)
	defer reader.Close()

	if err != nil {
		return nil, 0, fmt.Errorf("failed to open %s: %s", path, err)
	}

	buffer, err := io.ReadAll(reader)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to read %s: %s", path, err)
	}

	return buffer, fileInfo.Size(), nil
}

// ParseFileSize parses the GET_FILE_INFO response (ASCII decimal file size).
func ParseFileSize(response []byte) (int64, error) {
	raw := string(bytes.Trim(response, "\x00"))
	size, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid file size response: %q: %w", raw, err)
	}
	return size, nil
}

func ParsePath(path string) string {
	fileName := filepath.Base(path)
	newDir := "/root/video"
	return filepath.Join(newDir, fileName)
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

func Hertz(s float64, showUnit bool) string {
	sizes := []string{"MHz", "GHz", "THz", "PHz", "EHz"}
	return humanateHertz(s, 1000, sizes, showUnit)
}

func BitsShort(s uint64, showUnit bool) string {
	sizes := []string{"b", "k", "M", "G", "T", "P", "E"}
	return humanateBytes(s, 1000, sizes, showUnit)
}

func Bytes(s uint64, showUnit bool) string {
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(s, 1000, sizes, showUnit)
}

// NetSpeed formats a network speed value (bytes per second) into human-readable
// format like "1.2 MB/s", "950 KB/s", "0 B/s".
// Always uses format: value + space + unit, with consistent width.
func NetSpeed(bytesPerSec float64, showUnit bool) string {
	sizes := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	if bytesPerSec < 1 {
		if showUnit {
			return fmt.Sprintf("%3.f %s", 0.0, sizes[0])
		}
		return fmt.Sprintf("%3.f", 0.0)
	}
	e := math.Floor(logn(bytesPerSec, 1000))
	if int(e) >= len(sizes) {
		e = float64(len(sizes) - 1)
	}
	suffix := sizes[int(e)]
	val := math.Floor(bytesPerSec/math.Pow(1000, e)*10+0.5) / 10
	if showUnit {
		if val < 10 {
			return fmt.Sprintf("%3.1f %s", val, suffix)
		}
		return fmt.Sprintf("%3.f %s", val, suffix)
	}
	if val < 10 {
		return fmt.Sprintf("%3.1f", val)
	}
	return fmt.Sprintf("%3.f", val)
}

func humanateHertz(s float64, base float64, sizes []string, showUnit bool) string {
	if s < 10 {
		if showUnit {
			return fmt.Sprintf("%3.2f%s", s, sizes[0])
		}
		return fmt.Sprintf("%3.2f", s)
	}
	e := math.Floor(logn(s, base))
	suffix := sizes[int(e)]
	val := math.Floor(s/math.Pow(base, e)*10) / 10
	f := "%5.2f"
	if CountStr(suffix) == 3 {
		f = "%4.2f"
		if val < 10 {
			f = "%3.2f"
		}
	}
	if val < 10 {
		f = "%4.2f"
	}
	if showUnit {
		f += "%s"
		return fmt.Sprintf(f, val, suffix)
	}
	return fmt.Sprintf(f, val)
}

func humanateBytes(s uint64, base float64, sizes []string, showUnit bool) string {
	if s < 10 {
		if showUnit {
			return fmt.Sprintf("%5d%s", s, sizes[0])
		}
		return fmt.Sprintf("%5d", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%5.f"
	if CountStr(suffix) == 2 {
		f = "%4.f"
		if val < 10 {
			f = "%3.1f"
		}
	}
	if val < 10 {
		f = "%4.1f"
	}
	if showUnit {
		f += "%s"
		return fmt.Sprintf(f, val, suffix)
	}
	return fmt.Sprintf(f, val)
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

// CopyFile copia um arquivo do source para o destination.
func CopyFile(src, dst string) error {
	// If src and dst refer to the same file, skip to avoid truncating
	if src == dst {
		return nil
	}

	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	dstStat, err := os.Stat(dst)
	if err == nil && os.SameFile(srcStat, dstStat) {
		return nil
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// ParseColor converte uma string "r,g,b" para um objeto color.Color
func ParseColor(colorStr string) color.Color {
	if colorStr == "" {
		return color.White // Retorna branco se a cor não estiver definida
	}
	parts := strings.Split(colorStr, ",")
	if len(parts) != 3 {
		return color.White
	}
	r, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	g, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	b, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

// FormatColor converte um objeto color.Color para nossa string "r,g,b"
func FormatColor(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("%d,%d,%d", uint8(r), uint8(g), uint8(b))
}
