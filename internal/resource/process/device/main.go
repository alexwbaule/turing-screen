package device

import (
	"bytes"
	"image"
	"image/draw"
	"math/big"

	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/utils"
	"github.com/disintegration/imaging"
)

type ImageProcess struct {
	img image.Image
}

type ImageBackground interface {
	GetImage() image.Image
	GenerateBackgroundImage(orietation theme.Orientation) []byte
}

type ImagePartial interface {
	GetImage() image.Image
	GeneratePartialImage(orietation theme.Orientation, display *device.Display, xi, yi int) []byte
	TransformForOverlay(orientation theme.Orientation, display *device.Display, xi, yi int) (image.Image, int, int)
}

func NewImageProcess(i image.Image) *ImageProcess {
	return &ImageProcess{
		img: i,
	}
}

func (i *ImageProcess) GetImage() image.Image {
	return i.img
}

// NewScaledNRGBA scales src to dstW x dstH using CatmullRom interpolation.
func NewScaledNRGBA(src image.Image, dstW, dstH int) *image.NRGBA {
	rect := image.Rect(0, 0, dstW, dstH)
	dst := image.NewNRGBA(rect)
	draw.Draw(dst, rect, src, src.Bounds().Min, draw.Src)

	return dst
}

func (i *ImageProcess) GenerateBackgroundImage(orietation theme.Orientation) []byte {
	img := i.img
	if orietation == theme.REVERSE_LANDSCAPE {
		img = imaging.Rotate180(img)
	} else if orietation == theme.REVERSE_PORTRAIT {
		img = imaging.Rotate270(img)
	} else if orietation == theme.PORTRAIT {
		img = imaging.Rotate90(img)
	}

	bounds := img.Bounds()
	var imageAs bytes.Buffer
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			imageAs.Write([]byte{byte(b >> 8), byte(g >> 8), byte(r >> 8), byte(a >> 8)})
		}
	}
	return imageAs.Bytes()
}

// TransformForOverlay converts a builder-produced partial image and its
// theme-space coordinates into overlay display-space for draw.Draw().
//
// Unlike GeneratePartialImage (which encodes positions as byte offsets
// (x0+h)*W+y0 in the framebuffer), the overlay uses draw.Draw(x, y)
// where x=column, y=row directly on the display-space buffer. The
// coordinate systems are different, so the transformations differ.
//
// When display.Reverse=false and LANDSCAPE: the builder canvas is already
// 800x480 (display space), matching the overlay. No transformation needed.
func (i *ImageProcess) TransformForOverlay(orientation theme.Orientation, display *device.Display, x, y int) (image.Image, int, int) {
	img := i.img
	x0, y0 := x, y

	if orientation == theme.PORTRAIT {
		img = imaging.Rotate90(img)
		x0 = display.Height - x - img.Bounds().Dy()
	} else if orientation == theme.REVERSE_PORTRAIT {
		img = imaging.Rotate270(img)
		y0 = display.Width - y - img.Bounds().Dx()
	} else if orientation == theme.REVERSE_LANDSCAPE {
		img = imaging.Rotate180(img)
		y0 = display.Height - x - img.Bounds().Dx()
		x0 = display.Width - y - img.Bounds().Dy()
	} else if orientation == theme.LANDSCAPE {
		if display.Reverse {
			// Builder creates WxH (480x800 fb), overlay is HxW (800x480 display).
			// Swap coordinates to map fb→display.
			x0, y0 = y, x
		}
		// else: builder canvas IS display-sized (800x480), no transform needed.
	}

	return img, x0, y0
}

func (i *ImageProcess) GeneratePartialImage(orientation theme.Orientation, display *device.Display, x, y int) []byte {
	var imageAs bytes.Buffer

	img := i.img
	bounds := img.Bounds()
	x0, y0 := x, y

	if orientation == theme.PORTRAIT {
		img = imaging.Rotate90(img)
		x0 = display.Height - x - img.Bounds().Dy()
	} else if orientation == theme.REVERSE_PORTRAIT {
		img = imaging.Rotate270(img)
		y0 = display.Width - y - img.Bounds().Dx()
	} else if orientation == theme.REVERSE_LANDSCAPE {
		img = imaging.Rotate180(img)
		y0 = display.Height - x - img.Bounds().Dx()
		x0 = display.Width - y - img.Bounds().Dy()
	} else if orientation == theme.LANDSCAPE {
		x0, y0 = y, x
	}
	bounds = img.Bounds()

	for h := bounds.Min.Y; h < bounds.Max.Y; h++ {
		position := (x0+h)*display.Width + y0
		pwidth := img.Bounds().Size().X
		positions := make([]byte, 5)
		copy(positions, utils.PadBegin(big.NewInt(int64(position)).Bytes(), 3))
		copy(positions[3:], utils.PadBegin(big.NewInt(int64(pwidth)).Bytes(), 2))

		imageAs.Write(positions)
		for w := bounds.Min.X; w < bounds.Max.X; w++ {
			r, g, b, _ := img.At(w, h).RGBA()
			imageAs.Write([]byte{byte(b >> 8), byte(g >> 8), byte(r >> 8)})
		}
	}
	imageAs.Write([]byte{0xef, 0x69})
	return imageAs.Bytes()
}
