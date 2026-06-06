package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"github.com/alexwbaule/turing-screen/internal/theme"
	"github.com/alexwbaule/turing-screen/internal/utils"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// renderGGText renders text using Go stdlib (no gg dependency).
func renderGGText(data *theme.Text, fc *FontCache) image.Image {
	fontSize := data.FontSize
	if fontSize <= 0 {
		fontSize = 14
	}

	displayText := data.Text
	face := loadFace(data.Font, fontSize, fc)
	if face == nil {
		// Try default font as fallback
		face = loadFace("jetbrains-mono/JetBrainsMono-Bold.ttf", fontSize, fc)
		if face == nil {
			// Fallback: return a small placeholder
			img := image.NewRGBA(image.Rect(0, 0, 50, 20))
			return img
		}
	}

	// Measure using placeholder string of "8"s (same width as turing-screen daemon)
	measure := strings.Repeat("8", len([]rune(displayText)))
	d := &font.Drawer{Face: face}
	maxWidth := d.MeasureString(measure).Ceil()
	metrics := face.Metrics()
	height := (metrics.Ascent + metrics.Descent).Ceil()

	if maxWidth <= 0 {
		maxWidth = 50
	}
	if height <= 0 {
		height = 20
	}

	// Create image with background color or transparent
	var bgColor color.Color = color.Transparent
	if data.BackgroundColor != "" {
		bgColor = utils.ParseColor(data.BackgroundColor)
	}

	img := image.NewRGBA(image.Rect(0, 0, maxWidth, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	// Calculate text position based on alignment
	textWidth := d.MeasureString(displayText)
	var dotX fixed.Int26_6

	align := strings.ToUpper(data.Align)
	switch align {
	case "CENTER":
		dotX = (fixed.I(maxWidth) - textWidth) / 2
	case "RIGHT":
		dotX = fixed.I(maxWidth) - textWidth
	default: // LEFT
		dotX = 0
	}

	// Draw text
	fontColor := utils.ParseColor(data.FontColor)
	if fontColor == nil {
		fontColor = color.White
	}

	d.Dst = img
	d.Src = image.NewUniform(fontColor)
	d.Dot = fixed.Point26_6{X: dotX, Y: metrics.Ascent}
	d.DrawString(displayText)

	return img
}

// renderGGGraph renders a progress bar using Go stdlib.
// Preview renders at 75% fill.
func renderGGGraph(data *theme.Graph) image.Image {
	imgW := data.Width
	imgH := data.Height
	if imgW <= 0 {
		imgW = 100
	}
	if imgH <= 0 {
		imgH = 15
	}

	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.Transparent}, image.Point{}, draw.Src)

	// Preview value: 75%
	previewValue := 75.0
	maxVal := float64(data.MaxValue - data.MinValue)
	if maxVal <= 0 {
		maxVal = 100
	}
	barFilledWidth := int(math.Round(previewValue / maxVal * float64(imgW)))

	// Fill the bar
	barColor := utils.ParseColor(data.BarColor)
	if barColor == nil {
		barColor = color.White
	}
	fillRect(img, 0, 0, barFilledWidth, imgH, barColor)

	// Outline if requested
	if data.BarOutline {
		drawRectOutline(img, 0, 0, imgW, imgH, barColor)
	}

	return img
}

// renderGGRadial renders a radial arc using Go stdlib.
// Preview renders at 75%.
func renderGGRadial(data *theme.Radial, fc *FontCache) image.Image {
	diameter := data.Radius * 2
	if diameter <= 0 {
		diameter = 80
	}

	// Create NRGBA image (supports proper alpha/transparency)
	img := image.NewNRGBA(image.Rect(0, 0, diameter, diameter))
	// Leave fully transparent (zero-value NRGBA is transparent)

	// If background color is set, fill with it
	if data.BackgroundColor != "" {
		bgColor := utils.ParseColor(data.BackgroundColor)
		if bgColor != nil {
			draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)
		}
	}

	cx, cy := float64(data.Radius), float64(data.Radius)

	amin := radians(data.AngleStart)
	amax := radians(180 + data.AngleStart + data.AngleEnd)

	// Preview value: 75%
	previewValue := 75.0
	total := (previewValue * float64(180+data.AngleStart+data.AngleEnd)) / 100
	cur := radians(int(total) + data.AngleStart)
	if cur > amax {
		cur = amax
	}

	// Draw arc
	barColor := utils.ParseColor(data.BarColor)
	if barColor == nil {
		barColor = color.White
	}
	arcRadius := float64(data.Radius - data.Width/2)
	if arcRadius < 1 {
		arcRadius = 1
	}
	drawArcOnNRGBA(img, cx, cy, arcRadius, amin, cur, data.Width, barColor)

	// Draw text in center if ShowText
	if data.ShowText {
		fontSize := data.Radius / 3
		if fontSize <= 0 {
			fontSize = 10
		}
		face := loadFace(data.Font, fontSize, fc)
		if face != nil {
			measure := "75%"
			d := &font.Drawer{Face: face}
			textW := d.MeasureString(measure)
			metrics := face.Metrics()

			fontColor := utils.ParseColor(data.FontColor)
			if fontColor == nil {
				fontColor = color.White
			}

			dotX := fixed.I(data.Radius) - textW/2
			dotY := fixed.I(data.Radius) + metrics.Ascent/2 - metrics.Descent/2

			d.Dst = img
			d.Src = image.NewUniform(fontColor)
			d.Dot = fixed.Point26_6{X: dotX, Y: dotY}
			d.DrawString(measure)
		}
	}

	return img
}

// renderGGChart renders a bar chart preview using Go stdlib.
func renderGGChart(data *theme.Chart) image.Image {
	imgW := data.Width
	imgH := data.Height
	if imgW <= 0 {
		imgW = 100
	}
	if imgH <= 0 {
		imgH = 50
	}

	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.Transparent}, image.Point{}, draw.Src)

	colStep := data.ColumnWidth + data.ColumnGap
	if colStep <= 0 {
		colStep = 6
	}

	sampleValues := []float64{0.4, 0.7, 0.5, 0.8, 0.6, 0.3, 0.9, 0.55, 0.65, 0.45, 0.75, 0.35, 0.85, 0.5, 0.7, 0.6, 0.8, 0.4, 0.55, 0.9}

	fillColor := utils.ParseColor(data.FillColor)
	if fillColor == nil {
		fillColor = color.NRGBA{R: 0, G: 200, B: 100, A: 255}
	}

	for i, val := range sampleValues {
		x := i * colStep
		if x+data.ColumnWidth > imgW {
			break
		}
		barHeight := int(val * float64(imgH))
		y := imgH - barHeight
		fillRect(img, x, y, data.ColumnWidth, barHeight, fillColor)
	}

	// Border
	if data.BorderWidth > 0 {
		lineColor := utils.ParseColor(data.LineColor)
		if lineColor == nil {
			lineColor = color.White
		}
		drawRectOutline(img, 0, 0, imgW, imgH, lineColor)
	}

	return img
}

// --- Helper functions ---

func loadFace(fontPath string, fontSize int, fc *FontCache) font.Face {
	if fontPath == "" || fc == nil {
		return nil
	}
	face, err := fc.GetFace(fontPath, fontSize)
	if err != nil {
		return nil
	}
	return face
}

func radians(degrees int) float64 {
	return float64(degrees) * (math.Pi / 180.0)
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	for py := y; py < y+h && py < img.Bounds().Dy(); py++ {
		for px := x; px < x+w && px < img.Bounds().Dx(); px++ {
			if px >= 0 && py >= 0 {
				img.Set(px, py, c)
			}
		}
	}
}

func drawRectOutline(img *image.RGBA, x, y, w, h int, c color.Color) {
	for px := x; px < x+w; px++ {
		img.Set(px, y, c)
		img.Set(px, y+h-1, c)
	}
	for py := y; py < y+h; py++ {
		img.Set(x, py, c)
		img.Set(x+w-1, py, c)
	}
}

func drawArcStdlib(img *image.RGBA, cx, cy, radius, startAngle, endAngle float64, width int, c color.Color) {
	steps := int(math.Abs(endAngle-startAngle) * radius * 2)
	if steps < 100 {
		steps = 100
	}
	halfW := float64(width) / 2.0

	for i := 0; i <= steps; i++ {
		t := startAngle + (endAngle-startAngle)*float64(i)/float64(steps)
		px := cx + radius*math.Cos(t)
		py := cy + radius*math.Sin(t)

		for dy := -halfW; dy <= halfW; dy++ {
			for dx := -halfW; dx <= halfW; dx++ {
				if dx*dx+dy*dy <= halfW*halfW {
					ix := int(math.Round(px + dx))
					iy := int(math.Round(py + dy))
					if ix >= 0 && iy >= 0 && ix < img.Bounds().Dx() && iy < img.Bounds().Dy() {
						img.Set(ix, iy, c)
					}
				}
			}
		}
	}
}

func drawArcOnNRGBA(img *image.NRGBA, cx, cy, radius, startAngle, endAngle float64, width int, c color.Color) {
	steps := int(math.Abs(endAngle-startAngle) * radius * 2)
	if steps < 100 {
		steps = 100
	}
	halfW := float64(width) / 2.0

	for i := 0; i <= steps; i++ {
		t := startAngle + (endAngle-startAngle)*float64(i)/float64(steps)
		px := cx + radius*math.Cos(t)
		py := cy + radius*math.Sin(t)

		for dy := -halfW; dy <= halfW; dy++ {
			for dx := -halfW; dx <= halfW; dx++ {
				if dx*dx+dy*dy <= halfW*halfW {
					ix := int(math.Round(px + dx))
					iy := int(math.Round(py + dy))
					if ix >= 0 && iy >= 0 && ix < img.Bounds().Dx() && iy < img.Bounds().Dy() {
						img.SetNRGBA(ix, iy, colorToNRGBA(c))
					}
				}
			}
		}
	}
}

func colorToNRGBA(c color.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}
