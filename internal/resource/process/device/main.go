package device

import (
	"bytes"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/disintegration/imaging"
	"image"
	"math/big"
)

type ImageProcess struct {
	img image.Image
}

type ImageBackground interface {
	GenerateBackgroundImage(orietation theme.Orientation) []byte
}

type ImagePartial interface {
	GeneratePartialImage(orietation theme.Orientation, display *device.Display, xi, yi int) []byte
}

func NewImageProcess(i image.Image) *ImageProcess {
	return &ImageProcess{
		img: i,
	}
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
