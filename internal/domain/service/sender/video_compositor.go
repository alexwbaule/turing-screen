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

	// Match PIL paste(image, (x, y)): replace the destination pixels.
	drawRect := image.Rect(y0, x0, y0+img.Bounds().Dx(), x0+img.Bounds().Dy())
	draw.Draw(vc.current, drawRect, img, img.Bounds().Min, draw.Over)
}

// GenerateUpdate generates the img_raw_data and visible_pixels byte array
func (vc *VideoCompositor) GenerateUpdate(encoding command.PixelEncoding) ([]byte, []byte) {
	var imgRawData bytes.Buffer
	var visiblePixels bytes.Buffer

	bounds := vc.current.Bounds()
	logicalWidth := bounds.Dx()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		// Find updated visible segments
		x := bounds.Min.X
		for x < bounds.Max.X {
			currIdx := (y-bounds.Min.Y)*vc.current.Stride + (x-bounds.Min.X)*4
			prevIdx := (y-bounds.Min.Y)*vc.previous.Stride + (x-bounds.Min.X)*4

			changed := !bytes.Equal(vc.current.Pix[currIdx:currIdx+4], vc.previous.Pix[prevIdx:prevIdx+4])
			visible := vc.current.Pix[currIdx+3] > 0

			if changed && visible {
				startX := x
				width := 1
				x++
				for x < bounds.Max.X {
					cIdx := (y-bounds.Min.Y)*vc.current.Stride + (x-bounds.Min.X)*4
					pIdx := (y-bounds.Min.Y)*vc.previous.Stride + (x-bounds.Min.X)*4

					segmentChanged := !bytes.Equal(vc.current.Pix[cIdx:cIdx+4], vc.previous.Pix[pIdx:pIdx+4])
					segmentVisible := vc.current.Pix[cIdx+3] > 0

					if segmentChanged && segmentVisible {
						width++
						x++
					} else {
						break
					}
				}

				// Append to imgRawData
				position := (y-bounds.Min.Y)*logicalWidth + (startX - bounds.Min.X)
				positions := make([]byte, 5)
				copy(positions, utils.PadBegin(big.NewInt(int64(position)).Bytes(), 3))
				copy(positions[3:], utils.PadBegin(big.NewInt(int64(width)).Bytes(), 2))
				imgRawData.Write(positions)

				for w := 0; w < width; w++ {
					idx := (y-bounds.Min.Y)*vc.current.Stride + (startX+w-bounds.Min.X)*4
					r := vc.current.Pix[idx]
					g := vc.current.Pix[idx+1]
					b := vc.current.Pix[idx+2]
					a := vc.current.Pix[idx+3]

					r4 := byte(int(r) * 15 / 255)
					g4 := byte(int(g) * 15 / 255)
					b4 := byte(int(b) * 15 / 255)
					a4 := byte(int(a) * 15 / 255)

					bByte := (b4 << 4) | ((a4 & 0xC) >> 2)
					gByte := (g4 << 4) | (a4 & 0x3)
					rByte := r4 << 4

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

				position := (y-bounds.Min.Y)*logicalWidth + (startX - bounds.Min.X)
				positions := make([]byte, 5)
				copy(positions, utils.PadBegin(big.NewInt(int64(position)).Bytes(), 3))
				copy(positions[3:], utils.PadBegin(big.NewInt(int64(width)).Bytes(), 2))
				visiblePixels.Write(positions)
			} else {
				x++
			}
		}
	}

	// Match Python: previous_video_overlay = video_overlay.copy()
	copy(vc.previous.Pix, vc.current.Pix)

	return imgRawData.Bytes(), visiblePixels.Bytes()
}
