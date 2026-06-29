package compositor

import (
	"image"
	"image/color"
	"testing"
)

// TestDrawArcFullCircle ensures a 2π arc (a 0–360° radial background) fills the
// whole ring instead of collapsing to a single point under the [0,2π) modulo.
func TestDrawArcFullCircle(t *testing.T) {
	const size = 101
	const cx, cy = 50.0, 50.0
	const radius = 40.0
	const width = 8

	dst := image.NewNRGBA(image.Rect(0, 0, size, size))

	// Exactly one revolution: start and end differ by 2π. Before the fix, both
	// normalized to the same angle and the arc drew nothing.
	drawArcNRGBA(dst, cx, cy, radius, -1.5707963, -1.5707963+2*3.1415926, width, color.NRGBA{255, 0, 0, 255})

	want := color.NRGBA{255, 0, 0, 255}
	filled := 0
	// Sample a horizontal diameter through the center: every pixel in the band
	// must be the arc color for a full circle.
	for px := 0; px < size; px++ {
		c := dst.NRGBAAt(px, int(cy))
		if c == want {
			filled++
		}
	}

	// Ring spans roughly width pixels on each side of the diameter → expect
	// ~width filled pixels. A collapsed arc yields ~0–1.
	if filled < width-2 {
		t.Fatalf("full-circle arc only filled %d px across the diameter (want >= %d); the 2π case collapsed", filled, width-2)
	}

	// And the center must NOT be colored (it's inside the ring hole).
	if dst.NRGBAAt(int(cx), int(cy)) == want {
		t.Fatal("full-circle arc colored the ring hole — inner radius wrong")
	}
}
