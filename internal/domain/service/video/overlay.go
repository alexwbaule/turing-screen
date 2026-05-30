package video

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
)

// OverlayBuffer manages the video overlay buffers and diff computation.
type OverlayBuffer struct {
	mu       sync.Mutex
	log      *logger.Logger
	width    int
	height   int
	stride   int
	current  *image.NRGBA
	previous *image.NRGBA
	count    int64
}

// NewOverlayBuffer creates a new overlay buffer for the given display dimensions.
func NewOverlayBuffer(log *logger.Logger, display *device.Display) *OverlayBuffer {
	w := display.Width
	h := display.Height
	// In landscape mode, swap dimensions
	if display.Reverse {
		w, h = h, w
	}
	rect := image.Rect(0, 0, w, h)

	overlay := &OverlayBuffer{
		log:    log,
		width:  w,
		height: h,
		stride: w, // used for position encoding: hy*stride+x
	}

	overlay.current = image.NewNRGBA(rect)
	overlay.previous = image.NewNRGBA(rect)

	return overlay
}

func (o *OverlayBuffer) Width() int { return o.width }
func (o *OverlayBuffer) Height() int { return o.height }


// SetInitial sets the initial background image on both current and previous buffers.
// It copies pixels directly to NRGBA, ensuring alpha is preserved correctly.
func (o *OverlayBuffer) SetInitial(img image.Image) {
	o.mu.Lock()
	defer o.mu.Unlock()

	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	dstW, dstH := o.width, o.height

	// Determine copy region
	copyW := srcW
	copyH := srcH
	if copyW > dstW {
		copyW = dstW
	}
	if copyH > dstH {
		copyH = dstH
	}

	for y := 0; y < copyH; y++ {
		for x := 0; x < copyW; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// RGBA() returns 16-bit values, shift right 8 to get 8-bit
			o.current.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
			o.previous.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}
}

// Draw draws an image onto the current buffer at the given position.
// Thread-safe: can be called from multiple sensor goroutines.
func (o *OverlayBuffer) Draw(img image.Image, x, y int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	draw.Draw(o.current, img.Bounds().Add(image.Pt(x, y)), img, image.Point{}, draw.Src)
}

// Refresh computes the diff between current and previous buffers,
// generates the VideoOverlayRefresh command payload, and swaps the buffers.
func (o *OverlayBuffer) Refresh() *command.VideoOverlayRefresh {
	o.mu.Lock()
	defer o.mu.Unlock()

	w, h := o.width, o.height

	// Build diff image
	updateImage := image.NewNRGBA(image.Rect(0, 0, w, h))
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			c1 := o.previous.NRGBAAt(px, py)
			c2 := o.current.NRGBAAt(px, py)
			if c1 != c2 {
				updateImage.SetNRGBA(px, py, c2)
			}
		}
	}

	var imgRawData []string
	var visiblePixels []string

	for hy := 0; hy < h; hy++ {
		updatedSegments := getVisibleSegments(updateImage, hy, w)
		visibleSegments := getVisibleSegments(o.current, hy, w)

		for _, segment := range updatedSegments {
			x := segment[0]
			segWidth := segment[1]
			imgRawData = append(imgRawData, fmt.Sprintf("%06x%04x", hy*o.stride+x, segWidth))

			for wOffset := 0; wOffset < segWidth; wOffset++ {
				c := o.current.NRGBAAt(x+wOffset, hy)
				alphaByte := int(float64(c.A) / 255.0 * 15.0)
				b := int(float64(c.B)/255.0*15.0)<<4 | ((alphaByte & 0xC) >> 2)
				g := int(float64(c.G)/255.0*15.0)<<4 | (alphaByte & 0x3)
				r := int(float64(c.R)/255.0*15.0) << 4
				imgRawData = append(imgRawData, fmt.Sprintf("%02x%02x%02x", b, g, r))
			}
		}

		for _, segment := range visibleSegments {
			x := segment[0]
			segWidth := segment[1]
			visiblePixels = append(visiblePixels, fmt.Sprintf("%06x%04x", hy*o.stride+x, segWidth))
		}
	}

	imageMsg := fmt.Sprintf("%s%s", stringsJoin(imgRawData, ""), stringsJoin(visiblePixels, ""))
	visiblePixelsMsg := stringsJoin(visiblePixels, "")
	visiblePixelsSize := len(visiblePixelsMsg) / 2

	var imgPayload []byte
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

	o.log.Debugf("video overlay refresh: image_size=0x%06x vpSize=%d payload=%dB count=%d",
		imageSize, visiblePixelsSize, len(imgPayload), o.count)

	// Build the header (18 bytes)
	// UPDATE_BITMAP [0xCC, 0xEF, 0x69, 0x00] + imageSize(3) + pad(3) + count(4) + vpSize(4)
	header := make([]byte, 18)
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
	binary.BigEndian.PutUint32(header[10:14], uint32(o.count))

	// vpSize as 4 bytes big-endian
	binary.BigEndian.PutUint32(header[14:18], uint32(visiblePixelsSize))

	o.count++

	// Swap buffers
	draw.Draw(o.previous, o.previous.Bounds(), o.current, image.Point{}, draw.Src)

	return command.NewVideoOverlayRefresh(o.log, header, imgPayload)
}

// getVisibleSegments detects visible pixel segments per line.
func getVisibleSegments(img *image.NRGBA, y, width int) [][]int {
	var segments [][]int
	i := 0
	for i < width {
		if img.NRGBAAt(i, y).A > 0 {
			seg := []int{i, 1}
			j := i + 1
			for j < width && img.NRGBAAt(j, y).A > 0 {
				seg[1]++
				j++
			}
			i = j
			segments = append(segments, seg)
		} else {
			i++
		}
	}
	return segments
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
