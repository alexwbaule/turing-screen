package compositor

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"

	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
)

// TestTheme2Render generates PNGs of theme "2"'s composition so the layering
// can be inspected visually. Run: go test ./internal/domain/service/compositor/ -run TestTheme2Render -v
// Inspect: /tmp/theme2_*.png
func TestTheme2Render(t *testing.T) {
	root := "../../../../" // compositor pkg -> repo root
	bgImg, err := loadImg(root + "res/themes/2/assets/image_1.png")
	if err != nil {
		t.Skipf("theme assets not available: %v", err)
	}
	layerImg, err := loadImg(root + "res/themes/2/assets/image_3.png")
	if err != nil {
		t.Skipf("theme assets not available: %v", err)
	}

	const W, H = 1280, 800
	// Backdrop: BACKGROUND image composited over transparent (as BuildBackgroundImage does).
	backdrop := image.NewNRGBA(image.Rect(0, 0, W, H))
	draw.Draw(backdrop, backdrop.Bounds(), bgImg, bgImg.Bounds().Min, draw.Over)

	// CPU PERCENTAGE radial from theme 2.
	var c Compositor
	radial := &theme.Radial{}
	radial.X, radial.Y = 448, 371
	radial.Radius = 289
	radial.Width = 110
	radial.MinValue, radial.MaxValue = 0, 100
	radial.AngleStart, radial.AngleEnd = 212, 60
	radial.Clockwise = true // matches res/themes/2/theme.yaml's CLOCKWISE: true
	radial.BarColor = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	drawRadial := func(f *image.NRGBA) { c.drawRadialOnFrame(f, 100, radial) }
	drawLayer := func(f *image.NRGBA) {
		r := image.Rect(-12, -22, -12+layerImg.Bounds().Dx(), -22+layerImg.Bounds().Dy())
		draw.Draw(f, r, layerImg, layerImg.Bounds().Min, draw.Over)
	}

	// Case A: radial only (sanity — confirms the radial draws).
	fA := image.NewNRGBA(image.Rect(0, 0, W, H))
	copy(fA.Pix, backdrop.Pix)
	drawRadial(fA)
	savePNG(t, "/tmp/theme2_A_radial_only.png", fA)

	// Case B: radial BEHIND LAYER_2 (LAYER_2 drawn on top) — the reported problem.
	fB := image.NewNRGBA(image.Rect(0, 0, W, H))
	copy(fB.Pix, backdrop.Pix)
	drawRadial(fB)
	drawLayer(fB)
	savePNG(t, "/tmp/theme2_B_radial_behind.png", fB)

	// Case C: radial IN FRONT of LAYER_2.
	fC := image.NewNRGBA(image.Rect(0, 0, W, H))
	copy(fC.Pix, backdrop.Pix)
	drawLayer(fC)
	drawRadial(fC)
	savePNG(t, "/tmp/theme2_C_radial_front.png", fC)

	// Case D: radial WITH a gradient, behind LAYER_2 — does the gradient path
	// render the arc, or make it disappear?
	radialG := &theme.Radial{}
	*radialG = *radial
	radialG.GradientColor = color.NRGBA{R: 255, G: 0, B: 0, A: 255} // red→blue vertical gradient
	drawRadialG := func(f *image.NRGBA) { c.drawRadialOnFrame(f, 100, radialG) }
	fD := image.NewNRGBA(image.Rect(0, 0, W, H))
	copy(fD.Pix, backdrop.Pix)
	drawRadialG(fD)
	drawLayer(fD)
	savePNG(t, "/tmp/theme2_D_radial_gradient_behind.png", fD)
	// and gradient-only (no layer) for comparison
	fE := image.NewNRGBA(image.Rect(0, 0, W, H))
	copy(fE.Pix, backdrop.Pix)
	drawRadialG(fE)
	savePNG(t, "/tmp/theme2_E_radial_gradient_only.png", fE)

	t.Log("wrote /tmp/theme2_{A,B,C,D,E}.png")
}

func loadImg(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func savePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Errorf("create %s: %v", path, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Errorf("encode %s: %v", path, err)
	}
}
