package image_process

import (
	"bytes"
	"fmt"
	"github.com/alexwbaule/turing-screen/utils"
	"image"
	"math/big"
)

type ImageProcess struct {
	img image.Image
}

type ImageBackground interface {
	GenerateBackgroundImage() []byte
}

type ImagePartial interface {
	GeneratePartialImage(xi, yi int) []byte
}

func NewImageProcess(i image.Image) *ImageProcess {
	return &ImageProcess{
		img: i,
	}
}

func (i *ImageProcess) GenerateBackgroundImage() []byte {
	bounds := i.img.Bounds()
	var imageAs bytes.Buffer
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := i.img.At(x, y).RGBA()
			imageAs.Write([]byte{byte(b >> 8), byte(g >> 8), byte(r >> 8), byte(a >> 8)})
		}
	}
	return imageAs.Bytes()
}

func (i *ImageProcess) GeneratePartialImage(xi, yi int) []byte {
	bounds := i.img.Bounds()
	var imageAs bytes.Buffer

	x0, y0 := yi, xi

	for h := bounds.Min.Y; h < bounds.Max.Y; h++ {
		position := (x0+h)*800 + y0
		pwidth := i.img.Bounds().Size().X
		positions := make([]byte, 5)
		copy(positions, utils.PadBegin(big.NewInt(int64(position)).Bytes(), 3))
		copy(positions[3:], utils.PadBegin(big.NewInt(int64(pwidth)).Bytes(), 2))

		fmt.Printf("[%d] + [%d] * 800 + %d [%d]\n", x0, h, y0, pwidth)

		imageAs.Write(positions)
		for w := bounds.Min.X; w < bounds.Max.X; w++ {
			r, g, b, _ := i.img.At(w, h).RGBA()
			imageAs.Write([]byte{byte(b >> 8), byte(g >> 8), byte(r >> 8)})
		}
	}
	imageAs.Write([]byte{0xef, 0x69})
	return imageAs.Bytes()
}
