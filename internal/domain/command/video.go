package command

import (
	"image"
	"image/draw"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	xdraw "golang.org/x/image/draw"
)

// InitVideoOverlay represents the full video overlay initialization sequence.
// It replicates the reference InitializeVideoOverlay flow:
//
//	PRE_UPDATE_BITMAP → START_DISPLAY_BITMAP → DISPLAY_BITMAP_ON_VIDEO →
//	SEND_PAYLOAD(generateFullImage) → INIT_VIDEO_OVERLAY → SEND_PAYLOAD(visiblePixels)
type InitVideoOverlay struct {
	name          string
	log           *logger.Logger
	packets       [][]byte
	overlayNRGBA *image.NRGBA // the scaled background NRGBA (shared with OverlayBuffer)
}

func NewInitVideoOverlay(log *logger.Logger, backgroundImage image.Image, w, h int, orientation theme.Orientation) *InitVideoOverlay {
	var packets [][]byte

	// 1. PRE_UPDATE_BITMAP [0x86, 0xEF, 0x69, 0x00, 0x00, 0x00, 0x01] padded to 250
	pkt := utils.BZero(250, 0x00)
	copy(pkt, []byte{0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01})
	packets = append(packets, pkt)

	// 2. START_DISPLAY_BITMAP [0x2c] padded to 250 with 0x2c padding
	pkt = utils.BZero(250, 0x2c)
	copy(pkt, []byte{0x2c})
	packets = append(packets, pkt)

	// 3. DISPLAY_BITMAP_ON_VIDEO [0xCA, 0xEF, 0x69, 0x00, 0x17, 0x70] padded to 250
	pkt = utils.BZero(250, 0x00)
	copy(pkt, []byte{0xca, 0xef, 0x69, 0x00, 0x17, 0x70})
	packets = append(packets, pkt)

	// 4. Create NRGBA and scale background (matching reference InitializeVideoOverlay)
	rect := image.Rect(0, 0, w, h)
	var overlay *image.NRGBA
	if backgroundImage != nil {
		overlay = image.NewNRGBA(rect)
		xdraw.CatmullRom.Scale(overlay, rect, backgroundImage, backgroundImage.Bounds(), draw.Over, nil)
		log.Debugf("InitVideoOverlay: scaled background %dx%d -> %dx%d",
			backgroundImage.Bounds().Dx(), backgroundImage.Bounds().Dy(), w, h)
	} else {
		overlay = image.NewNRGBA(rect)
		log.Debug("InitVideoOverlay: using transparent overlay")
	}

	// 5. Generate BGRA payload with 249+1 chunking (generateFullImage)
	bgraData := generateFullImage(overlay, orientation)

	// Split into 250-byte packets (generateFullImage already has 249+1 structure)
	for i := 0; i < len(bgraData); i += 250 {
		end := i + 250
		if end > len(bgraData) {
			end = len(bgraData)
		}
		pkt = utils.BZero(250, 0x00)
		copy(pkt, bgraData[i:end])
		packets = append(packets, pkt)
	}

	// 6. INIT_VIDEO_OVERLAY [0xD0, 0xEF, 0x69, 0x00, 0x00, 0x00, 0x02] padded to 250
	pkt = utils.BZero(250, 0x00)
	copy(pkt, []byte{0xd0, 0xef, 0x69, 0x00, 0x00, 0x00, 0x02})
	packets = append(packets, pkt)

	// 7. SEND_PAYLOAD with visible pixels [0xEF, 0x69] padded to 250
	pkt = utils.BZero(250, 0x00)
	copy(pkt, []byte{0xef, 0x69})
	packets = append(packets, pkt)

	log.Debugf("InitVideoOverlay: %d packets (3 cmds + %d payload + 2 init), bgra=%d bytes",
		len(packets), (len(bgraData)+249)/250, len(bgraData))

	return &InitVideoOverlay{
		name:          "INIT_VIDEO_OVERLAY",
		log:           log,
		packets:       packets,
		overlayNRGBA:  overlay,
	}
}

func (c *InitVideoOverlay) GetBytes() [][]byte {
	return c.packets
}

func (c *InitVideoOverlay) SetCount(count int64) {
	_ = count
}

func (c *InitVideoOverlay) GetName() string {
	return c.name
}

// OverlayNRGBA returns the scaled background NRGBA created during init.
// This should be passed to OverlayBuffer.SetInitial() so the refresh
// engine uses the same image that was sent to the device.
func (c *InitVideoOverlay) OverlayNRGBA() *image.NRGBA {
	return c.overlayNRGBA
}

func (c *InitVideoOverlay) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  0,
		Bytes: nil,
	}
}

func (c *InitVideoOverlay) ValidateCommand([]byte, int) error {
	return nil
}

// generateFullImage converts an image to BGRA bytes and splits into 249-byte
// chunks with 0x00 separators, matching the reference _generate_full_image from PR #348.
func generateFullImage(img image.Image, orientation theme.Orientation) []byte {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	var rotated *image.NRGBA
	switch orientation {
	case theme.PORTRAIT:
		rotated = image.NewNRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rotated.Set(y, w-1-x, img.At(x, y))
			}
		}
	case theme.REVERSE_PORTRAIT:
		rotated = image.NewNRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rotated.Set(h-1-y, x, img.At(x, y))
			}
		}
	case theme.REVERSE_LANDSCAPE:
		rotated = image.NewNRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rotated.Set(w-1-x, h-1-y, img.At(x, y))
			}
		}
	default: // LANDSCAPE
		rotated = image.NewNRGBA(bounds)
		draw.Draw(rotated, bounds, img, bounds.Min, draw.Src)
	}

	rw, rh := rotated.Bounds().Dx(), rotated.Bounds().Dy()
	var bgraData []byte
	for y := 0; y < rh; y++ {
		for x := 0; x < rw; x++ {
			c := rotated.NRGBAAt(x, y)
			bgraData = append(bgraData, c.B, c.G, c.R, c.A)
		}
	}

	var result []byte
	for i := 0; i < len(bgraData); i += 249 {
		end := i + 249
		if end > len(bgraData) {
			end = len(bgraData)
		}
		if i > 0 {
			result = append(result, 0x00)
		}
		result = append(result, bgraData[i:end]...)
	}

	return result
}
