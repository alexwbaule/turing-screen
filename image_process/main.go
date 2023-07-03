package image_process

import (
	"bytes"
	"github.com/alexwbaule/turing-screen/utils"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math/big"
	"os"
)

type ImageProcess struct {
	img image.Image
}

func NewImageProcess(i image.Image) *ImageProcess {
	return &ImageProcess{
		img: i,
	}
}

func LoadImage(path string) image.Image {
	reader, err := os.Open(path)
	if err != nil {
		log.Fatalf("failed to open: %s", err)
	}
	defer reader.Close()
	img, _, err := image.Decode(reader)
	if err != nil {
		log.Fatalf("failed to decode: %s", err)
	}
	log.Printf("%d x %d", img.Bounds().Size().X, img.Bounds().Size().Y)
	return img
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

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		position := (yi+y)*800 + xi
		pwidth := i.img.Bounds().Size().X
		positions := make([]byte, 5)
		copy(positions, utils.PadBegin(big.NewInt(int64(position)).Bytes(), 3))
		copy(positions[3:], utils.PadBegin(big.NewInt(int64(pwidth)).Bytes(), 2))

		imageAs.Write(positions)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := i.img.At(x, y).RGBA()
			imageAs.Write([]byte{byte(b >> 8), byte(g >> 8), byte(r >> 8)})
		}
	}
	imageAs.Write([]byte{0xef, 0x69})
	return imageAs.Bytes()
}
