package sender

import (
	"bytes"
	"image"
	"image/draw"
	"math/big"

	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	pdevice "github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/disintegration/imaging"
)

type VideoCompositor struct {
	display     *device.Display
	orientation theme.Orientation
	current     *image.RGBA
	previous    *image.RGBA
	baseBg      image.Image
}

func NewVideoCompositor(display *device.Display, orientation theme.Orientation, bg pdevice.ImageBackground) *VideoCompositor {
	var baseImg image.Image = bg.GetImage()
	if orientation == theme.REVERSE_LANDSCAPE {
		baseImg = imaging.Rotate180(baseImg)
	} else if orientation == theme.REVERSE_PORTRAIT {
		baseImg = imaging.Rotate270(baseImg)
	} else if orientation == theme.PORTRAIT {
		baseImg = imaging.Rotate90(baseImg)
	}

	bounds := baseImg.Bounds()
	current := image.NewRGBA(bounds)
	draw.Draw(current, bounds, baseImg, bounds.Min, draw.Src)

	previous := image.NewRGBA(bounds)
	draw.Draw(previous, bounds, baseImg, bounds.Min, draw.Src)

	return &VideoCompositor{
		display:     display,
		orientation: orientation,
		current:     current,
		previous:    previous,
		baseBg:      baseImg,
	}
}

// ApplyUpdate draws a partial update onto the current buffer.
func (vc *VideoCompositor) ApplyUpdate(up *command.UpdatePayload) {
	img := up.GetPartialImage()
	if img == nil {
		return
	}
	x, y := up.GetRegion().X, up.GetRegion().Y

	// Determine coordinates in the rotated space
	x0, y0 := x, y
	if vc.orientation == theme.PORTRAIT {
		img = imaging.Rotate90(img)
		x0 = vc.display.Height - x - img.Bounds().Dy()
	} else if vc.orientation == theme.REVERSE_PORTRAIT {
		img = imaging.Rotate270(img)
		y0 = vc.display.Width - y - img.Bounds().Dx()
	} else if vc.orientation == theme.REVERSE_LANDSCAPE {
		img = imaging.Rotate180(img)
		y0 = vc.display.Height - x - img.Bounds().Dx()
		x0 = vc.display.Width - y - img.Bounds().Dy()
	} else if vc.orientation == theme.LANDSCAPE {
		x0, y0 = y, x
	}

	// Draw the image onto the current buffer
	drawRect := image.Rect(y0, x0, y0+img.Bounds().Dx(), x0+img.Bounds().Dy())
	draw.Draw(vc.current, drawRect, img, img.Bounds().Min, draw.Over) // draw.Over for alpha blending
}

// GenerateUpdate generates the img_raw_data and visible_pixels byte array
func (vc *VideoCompositor) GenerateUpdate(encoding command.PixelEncoding) ([]byte, []byte) {
	var imgRawData bytes.Buffer
	var visiblePixels bytes.Buffer

	bounds := vc.current.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		// Find updated segments
		x := bounds.Min.X
		for x < bounds.Max.X {
			currIdx := (y-bounds.Min.Y)*vc.current.Stride + (x-bounds.Min.X)*4
			prevIdx := (y-bounds.Min.Y)*vc.previous.Stride + (x-bounds.Min.X)*4

			if !bytes.Equal(vc.current.Pix[currIdx:currIdx+4], vc.previous.Pix[prevIdx:prevIdx+4]) {
				startX := x
				width := 1
				x++
				for x < bounds.Max.X {
					cIdx := (y-bounds.Min.Y)*vc.current.Stride + (x-bounds.Min.X)*4
					pIdx := (y-bounds.Min.Y)*vc.previous.Stride + (x-bounds.Min.X)*4
					if !bytes.Equal(vc.current.Pix[cIdx:cIdx+4], vc.previous.Pix[pIdx:pIdx+4]) {
						width++
						x++
					} else {
						break
					}
				}

				// Append to imgRawData
				position := y*vc.display.Width + startX
				positions := make([]byte, 5)
				copy(positions, utils.PadBegin(big.NewInt(int64(position)).Bytes(), 3))
				copy(positions[3:], utils.PadBegin(big.NewInt(int64(width)).Bytes(), 2))
				imgRawData.Write(positions)

				for w := 0; w < width; w++ {
					// Pega a cor diretamente do buffer Pix (8 bits por canal, sem pre-multiplicação)
					idx := (y-bounds.Min.Y)*vc.current.Stride + (startX+w-bounds.Min.X)*4
					r := vc.current.Pix[idx]
					g := vc.current.Pix[idx+1]
					b := vc.current.Pix[idx+2]
					a := vc.current.Pix[idx+3]

					// Codificação especial para Video Overlay:
					// O dispositivo espera 3 bytes por pixel, mas precisa do canal Alpha.
					// Portanto, as cores são espremidas em 4 bits (descartando os 4 bits menos significativos)
					// e os 4 bits do Alfa são distribuídos nos bits livres.
					// Formato:
					// Byte 0: BBBB AA (top 2 bits do alpha)
					// Byte 1: GGGG aa (bottom 2 bits do alpha)
					// Byte 2: RRRR 0000
					
					alpha4bit := a >> 4
					bByte := (b & 0xF0) | ((alpha4bit & 0xC) >> 2)
					gByte := (g & 0xF0) | (alpha4bit & 0x3)
					rByte := (r & 0xF0)

					imgRawData.Write([]byte{bByte, gByte, rByte})
				}
			} else {
				x++
			}
		}

		// Find all visible segments (alpha > 0)
		x = bounds.Min.X
		for x < bounds.Max.X {
			idx := (y-bounds.Min.Y)*vc.current.Stride + (x-bounds.Min.X)*4
			if vc.current.Pix[idx+3] > 0 {
				startX := x
				width := 1
				x++
				for x < bounds.Max.X {
					idx2 := (y-bounds.Min.Y)*vc.current.Stride + (x-bounds.Min.X)*4
					if vc.current.Pix[idx2+3] > 0 {
						width++
						x++
					} else {
						break
					}
				}

				position := y*vc.display.Width + startX
				positions := make([]byte, 5)
				copy(positions, utils.PadBegin(big.NewInt(int64(position)).Bytes(), 3))
				copy(positions[3:], utils.PadBegin(big.NewInt(int64(width)).Bytes(), 2))
				visiblePixels.Write(positions)
			} else {
				x++
			}
		}
	}

	// Copy current to previous
	copy(vc.previous.Pix, vc.current.Pix)

	// Reset current to the base background for the next composition cycle
	draw.Draw(vc.current, bounds, vc.baseBg, bounds.Min, draw.Src)

	return imgRawData.Bytes(), visiblePixels.Bytes()
}