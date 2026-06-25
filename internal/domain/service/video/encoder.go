package video

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
)

// BuildInitPayload rotates an NRGBA image based on orientation and converts
// to BGRA bytes with 249+1 chunking, matching the reference _generate_full_image
// from PR #348.
func BuildInitPayload(img *image.NRGBA, orientation theme.Orientation) []byte {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	var rotated *image.NRGBA
	switch orientation {
	case theme.PORTRAIT:
		// 90° CCW: dest(col=H-1-y, row=x) — matches Python image.rotate(90)
		rotated = image.NewNRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rotated.Set(h-1-y, x, img.At(x, y))
			}
		}
	case theme.REVERSE_PORTRAIT:
		// 90° CW (270° CCW): dest(col=y, row=W-1-x) — matches Python image.rotate(270)
		rotated = image.NewNRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rotated.Set(y, w-1-x, img.At(x, y))
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

// OverlayEncoder handles the protocol-level encoding for video overlay refresh
// commands: 4-bit pixel packing, position encoding, boundary fix, and header
// construction.
type OverlayEncoder struct {
	log          *logger.Logger
	stride       int
	count        int64
	imgRawData   []string
	visiblePixel []string
}

// NewOverlayEncoder creates a new encoder for the given display stride (width).
func NewOverlayEncoder(log *logger.Logger, w int) *OverlayEncoder {
	return &OverlayEncoder{
		log:    log,
		stride: w,
	}
}

// ProcessRow encodes one scanline of update and visible segments with 4-bit
// pixel packing and position encoding.
func (e *OverlayEncoder) ProcessRow(hy int, updatedSegments, visibleSegments [][]int, current *image.NRGBA) {
	for _, segment := range updatedSegments {
		x := segment[0]
		segWidth := segment[1]
		e.imgRawData = append(e.imgRawData, fmt.Sprintf("%06x%04x", hy*e.stride+x, segWidth))

		for wOffset := 0; wOffset < segWidth; wOffset++ {
			c := current.NRGBAAt(x+wOffset, hy)
			alphaByte := int(float64(c.A) / 255.0 * 15.0)
			b := int(float64(c.B)/255.0*15.0)<<4 | ((alphaByte & 0xC) >> 2)
			g := int(float64(c.G)/255.0*15.0)<<4 | (alphaByte & 0x3)
			r := int(float64(c.R)/255.0*15.0) << 4
			e.imgRawData = append(e.imgRawData, fmt.Sprintf("%02x%02x%02x", b, g, r))
		}
	}

	for _, segment := range visibleSegments {
		x := segment[0]
		segWidth := segment[1]
		e.visiblePixel = append(e.visiblePixel, fmt.Sprintf("%06x%04x", hy*e.stride+x, segWidth))
	}
}

// Finalize assembles the accumulated row data into the refresh header (18 bytes)
// and payload, applying boundary fix when needed. Resets internal state for the
// next refresh cycle.
func (e *OverlayEncoder) Finalize() (header []byte, imgPayload []byte) {
	imageMsg := fmt.Sprintf("%s%s", stringsJoin(e.imgRawData, ""), stringsJoin(e.visiblePixel, ""))
	visiblePixelsMsg := stringsJoin(e.visiblePixel, "")
	visiblePixelsSize := len(visiblePixelsMsg) / 2

	var imageSize int

	if len(imageMsg) > 500 {
		var chunks []string
		for i := 0; i < len(imageMsg); i += 498 {
			end := i + 498
			if end > len(imageMsg) {
				end = len(imageMsg)
			}
			chunks = append(chunks, imageMsg[i:end])
		}
		chunkedMsg := stringsJoin(chunks, "00")
		imgPayloadBytes, _ := hexDecodeString(chunkedMsg)
		imgPayload = append(imgPayload, imgPayloadBytes...)

		mod := len(imgPayload) % 250
		if len(imgPayload) > 250 && (mod == 0 || mod == 248 || mod == 249) {
			// Boundary fix: inject extra bytes
			imgPayload = append(imgPayload, []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0xef, 0x69}...)
			imageSize = (len(imageMsg)/2) + 7
			visiblePixelsSize += 5
		} else {
			imgPayload = append(imgPayload, []byte{0xef, 0x69}...)
			imageSize = (len(imageMsg)/2) + 2
		}
	} else {
		if len(imageMsg) > 0 {
			imgPayloadBytes, _ := hexDecodeString(imageMsg)
			imgPayload = append(imgPayload, imgPayloadBytes...)
		}
		imgPayload = append(imgPayload, []byte{0xef, 0x69}...)
		imageSize = (len(imageMsg)/2) + 2
	}

	e.log.Debugf("video overlay refresh: image_size=0x%06x vpSize=%d payload=%dB count=%d",
		imageSize, visiblePixelsSize, len(imgPayload), e.count)

	// Build header (18 bytes)
	// UPDATE_BITMAP [0xCC, 0xEF, 0x69, 0x00] + imageSize(3) + pad(3) + count(4) + vpSize(4)
	header = make([]byte, 18)
	header[0] = 0xcc
	header[1] = 0xef
	header[2] = 0x69
	header[3] = 0x00

	// imageSize as 3 bytes big-endian
	header[4] = byte(imageSize >> 16)
	header[5] = byte(imageSize >> 8)
	header[6] = byte(imageSize)

	// pad(3) = 0x00
	header[7] = 0x00
	header[8] = 0x00
	header[9] = 0x00

	// count as 4 bytes big-endian
	binary.BigEndian.PutUint32(header[10:14], uint32(e.count))

	// vpSize as 4 bytes big-endian
	binary.BigEndian.PutUint32(header[14:18], uint32(visiblePixelsSize))

	e.count++

	// Reset accumulated data for next cycle
	e.imgRawData = nil
	e.visiblePixel = nil

	return header, imgPayload
}

// stringsJoin is a simple string joiner (avoids strings import in this package).
func stringsJoin(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	result := elems[0]
	for i := 1; i < len(elems); i++ {
		result += sep + elems[i]
	}
	return result
}

// hexDecodeString decodes a hex string to bytes.
func hexDecodeString(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("odd hex string length")
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(b); i++ {
		c1 := hexVal(s[2*i])
		c2 := hexVal(s[2*i+1])
		b[i] = (c1 << 4) | c2
	}
	return b, nil
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}
