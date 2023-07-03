package utils

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
)

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
