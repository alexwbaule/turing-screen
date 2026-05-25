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
			currR, curG, currB, currA := vc.current.At(x, y).RGBA()
			prevR, prevG, prevB, prevA := vc.previous.At(x, y).RGBA()

			if currR != prevR || curG != prevG || currB != prevB || currA != prevA {
				startX := x
				width := 1
				x++
				for x < bounds.Max.X {
					cR, cG, cB, cA := vc.current.At(x, y).RGBA()
					pR, pG, pB, pA := vc.previous.At(x, y).RGBA()
					if cR != pR || cG != pG || cB != pB || cA != pA {
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
					r, g, b, a := vc.current.At(startX+w, y).RGBA()
					if encoding == command.EncodingBGRA {
						imgRawData.Write([]byte{byte(b >> 8), byte(g >> 8), byte(r >> 8), byte(a >> 8)})
					} else {
						imgRawData.Write([]byte{byte(b >> 8), byte(g >> 8), byte(r >> 8)})
					}
				}
			} else {
				x++
			}
		}

		// Find all visible segments (alpha > 0)
		x = bounds.Min.X
		for x < bounds.Max.X {
			_, _, _, a := vc.current.At(x, y).RGBA()
			if a > 0 {
				startX := x
				width := 1
				x++
				for x < bounds.Max.X {
					_, _, _, a2 := vc.current.At(x, y).RGBA()
					if a2 > 0 {
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
	draw.Draw(vc.previous, bounds, vc.current, bounds.Min, draw.Src)

	return imgRawData.Bytes(), visiblePixels.Bytes()
}