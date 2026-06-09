package video

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
)

// OverlayBuffer manages the video overlay buffers and diff computation.
type OverlayBuffer struct {
	mu       sync.Mutex
	log      *logger.Logger
	width    int
	height   int
	current  *image.NRGBA
	previous *image.NRGBA
	encoder  *OverlayEncoder
}

// NewOverlayBuffer creates a new overlay buffer for the given display and orientation.
// The display dimensions in the config represent the effective display size
// (e.g. Width=800, Height=480 for LANDSCAPE), so no swap is needed.
func NewOverlayBuffer(log *logger.Logger, display *device.Display, orientation theme.Orientation) *OverlayBuffer {
	var w, h int
	switch orientation {
	case theme.LANDSCAPE, theme.REVERSE_LANDSCAPE:
		w = display.Width
		h = display.Height
	default: // PORTRAIT, REVERSE_PORTRAIT
		w = display.Height
		h = display.Width
	}
	rect := image.Rect(0, 0, w, h)

	overlay := &OverlayBuffer{
		log:     log,
		width:   w,
		height:  h,
		encoder: NewOverlayEncoder(log, w),
	}

	overlay.current = image.NewNRGBA(rect)
	overlay.previous = image.NewNRGBA(rect)

	return overlay
}

func (o *OverlayBuffer) Width() int  { return o.width }
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
// delegates encoding to the OverlayEncoder, and swaps the buffers.
func (o *OverlayBuffer) Refresh() command.Command {
	o.mu.Lock()
	defer o.mu.Unlock()

	w, h := o.width, o.height

	// Build diff image using direct Pix slice access for performance
	updateImage := image.NewNRGBA(image.Rect(0, 0, w, h))
	prev := o.previous
	cur := o.current
	stride := cur.Stride
	for py := 0; py < h; py++ {
		rowStart := py * stride
		for px := 0; px < w; px++ {
			off := rowStart + px*4
			if prev.Pix[off] != cur.Pix[off] ||
				prev.Pix[off+1] != cur.Pix[off+1] ||
				prev.Pix[off+2] != cur.Pix[off+2] ||
				prev.Pix[off+3] != cur.Pix[off+3] {
				copy(updateImage.Pix[off:off+4], cur.Pix[off:off+4])
			}
		}
	}

	// Segment detection and encoding delegation
	for hy := 0; hy < h; hy++ {
		updatedSegments := getVisibleSegments(updateImage, hy, w)
		visibleSegments := getVisibleSegments(o.current, hy, w)
		o.encoder.ProcessRow(hy, updatedSegments, visibleSegments, o.current)
	}

	// Encode refresh data (4-bit packing, boundary fix, header)
	header, imgPayload := o.encoder.Finalize()

	// Swap buffers
	draw.Draw(o.previous, o.previous.Bounds(), o.current, image.Point{}, draw.Src)

	return command.NewVideoOverlayRefresh(o.log, header, imgPayload)
}

// getVisibleSegments detects visible pixel segments per line.
func getVisibleSegments(img *image.NRGBA, y, width int) [][]int {
	var segments [][]int
	rowStart := y * img.Stride
	i := 0
	for i < width {
		// Alpha is at offset+3
		if img.Pix[rowStart+i*4+3] > 0 {
			seg := []int{i, 1}
			j := i + 1
			for j < width && img.Pix[rowStart+j*4+3] > 0 {
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
