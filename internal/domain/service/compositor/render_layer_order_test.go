package compositor

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
)

// TestRadialBehindLayerImage validates the daemon's image composition when a
// radial sits behind a layer image: the radial must remain visible through the
// layer's transparent pixels, and be hidden only where the layer is opaque.
func TestRadialBehindLayerImage(t *testing.T) {
	const W, H = 200, 200
	mkFrame := func() *image.NRGBA {
		f := image.NewNRGBA(image.Rect(0, 0, W, H))
		fillNRGBA(f, 0, 0, W, H, color.NRGBA{20, 20, 20, 255}) // opaque backdrop
		return f
	}
	var c Compositor

	// Radial centered at (100,100), radius 40, width 20 → arc band radius 20–40.
	// At the top (12 o'clock) the band covers pixel (100, 70) [distance 30].
	radial := &theme.Radial{}
	radial.X, radial.Y = 100, 100
	radial.Radius = 40
	radial.Width = 20
	radial.MinValue, radial.MaxValue = 0, 100
	radial.AngleStart, radial.AngleEnd = 0, 270
	radial.Clockwise = true
	radial.BarColor = color.NRGBA{R: 0, G: 255, B: 0, A: 255}

	probe := func(f *image.NRGBA) color.NRGBA { return f.NRGBAAt(100, 70) }

	// Sanity: radial alone draws green at the probe.
	f1 := mkFrame()
	c.drawRadialOnFrame(f1, 100, radial)
	if p := probe(f1); p.G < 200 {
		t.Fatalf("radial not drawn at probe: %+v", p)
	}

	// Case A: a FULLY TRANSPARENT layer image over the radial → radial must show.
	f2 := mkFrame()
	c.drawRadialOnFrame(f2, 100, radial)
	transparent := image.NewNRGBA(image.Rect(0, 0, W, H)) // alpha 0 everywhere
	draw.Draw(f2, f2.Bounds(), transparent, transparent.Bounds().Min, draw.Over)
	if p := probe(f2); p.G < 200 {
		t.Fatalf("transparent layer hid the radial (should show through): %+v", p)
	}

	// Case B: a FULLY OPAQUE layer image over the radial → radial must be hidden.
	f3 := mkFrame()
	c.drawRadialOnFrame(f3, 100, radial)
	opaque := image.NewNRGBA(image.Rect(0, 0, W, H))
	fillNRGBA(opaque, 0, 0, W, H, color.NRGBA{255, 0, 0, 255}) // opaque red
	draw.Draw(f3, f3.Bounds(), opaque, opaque.Bounds().Min, draw.Over)
	if p := probe(f3); p.G > 50 || p.R < 200 {
		t.Fatalf("opaque layer did not cover the radial: %+v", p)
	}
}
