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
	maxWidth := d.MeasureString(measure).Ceil() + imageRightPad
	metrics := face.Metrics()
	height := (metrics.Ascent + metrics.Descent).Ceil()

	if maxWidth <= imageRightPad {
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

	previewRatio := 0.75
	if data.RevertValue {
		previewRatio = 1 - previewRatio
	}

	barColor := utils.ParseColor(data.BarColor)
	if barColor == nil {
		barColor = color.White
	}
	var bgColor color.Color
	if data.BackgroundColor != "" {
		bgColor = utils.ParseColor(data.BackgroundColor)
	}
	var gradientColor color.Color
	if data.GradientColor != "" {
		gradientColor = utils.ParseColor(data.GradientColor)
	}
	radius := data.CornerRadius
	hasGradient := gradientColor != nil
	hasRound := radius > 0

	fillBarRGBA := func(x, y, w, h int, clr color.Color) {
		if w <= 0 || h <= 0 || clr == nil {
			return
		}
		if hasGradient && hasRound {
			fillGradientRoundedRGBA(img, x, y, w, h, radius, gradientColor, clr)
		} else if hasGradient {
			fillGradientRGBA(img, x, y, w, h, gradientColor, clr)
		} else if hasRound {
			fillRoundedRGBA(img, x, y, w, h, radius, clr)
		} else {
			fillRect(img, x, y, w, h, clr)
		}
	}

	filledW := int(math.Round(previewRatio * float64(imgW)))

	// Compute effective steps
	steps := data.Steps
	if data.BlockWidth > 0 {
		gap := data.StepGap
		if gap < 0 {
			gap = 0
		}
		steps = imgW / (data.BlockWidth + gap)
		if steps < 1 {
			steps = 1
		}
	}

	if steps > 0 && data.StepGap > 0 {
		gap := data.StepGap
		segW := data.BlockWidth
		if segW <= 0 {
			segW = (imgW - (steps-1)*gap) / steps
		}
		if segW < 1 {
			segW = 1
		}
		filledSteps := int(math.Round(previewRatio * float64(steps)))
		for i := 0; i < steps; i++ {
			sx := i * (segW + gap)
			if i < filledSteps {
				fillBarRGBA(sx, 0, segW, imgH, barColor)
			} else {
				fillBarRGBA(sx, 0, segW, imgH, bgColor)
			}
		}
	} else {
		fillBarRGBA(filledW, 0, imgW-filledW, imgH, bgColor)
		fillBarRGBA(0, 0, filledW, imgH, barColor)
	}

	if data.BorderWidth > 0 {
		for i := 0; i < data.BorderWidth; i++ {
			drawRectOutline(img, i, i, imgW-2*i, imgH-2*i, barColor)
		}
	} else if data.BarOutline {
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

	img := image.NewNRGBA(image.Rect(0, 0, diameter, diameter))

	cx, cy := float64(data.Radius), float64(data.Radius)

	// Offset by -90° so AngleStart=0 → 12 o'clock (top).
	amin := radians(data.AngleStart - 90)
	amax := radians(180 + data.AngleStart - 90 + data.AngleEnd)
	totalArc := amax - amin

	previewRatio := 0.75
	filledAngle := totalArc * previewRatio

	var cur float64
	if data.RevertValue {
		cur = amax - filledAngle
	} else {
		cur = amin + filledAngle
	}
	if cur > amax {
		cur = amax
	}

	barColor := utils.ParseColor(data.BarColor)
	if barColor == nil {
		barColor = color.White
	}
	var emptyColor color.Color
	if data.BackgroundColor != "" {
		if c := utils.ParseColor(data.BackgroundColor); c != nil {
			if _, _, _, a := c.RGBA(); a > 0 {
				emptyColor = c
			}
		}
	}
	var gradientColor color.Color
	if data.GradientColor != "" {
		gradientColor = utils.ParseColor(data.GradientColor)
	}

	if data.Revert {
		barColor, emptyColor = emptyColor, barColor
	}

	arcRadius := float64(data.Radius - data.Width/2)
	if arcRadius < 1 {
		arcRadius = 1
	}

	drawArcFn := func(a1, a2 float64, clr color.Color) {
		if clr == nil || a2 <= a1 {
			return
		}
		if gradientColor != nil {
			drawArcGradientOnNRGBA(img, cx, cy, arcRadius, a1, a2, data.Width, gradientColor, clr)
		} else {
			drawArcOnNRGBA(img, cx, cy, arcRadius, a1, a2, data.Width, clr)
		}
		if data.Round {
			r := float64(data.Width) / 2.0
			drawCircleOnNRGBA(img, cx+arcRadius*math.Cos(a1), cy+arcRadius*math.Sin(a1), r, clr)
			drawCircleOnNRGBA(img, cx+arcRadius*math.Cos(a2), cy+arcRadius*math.Sin(a2), r, clr)
		}
	}

	// Determine step count (BlockAngle takes priority)
	angleSteps := data.AngleSteps
	if data.BlockAngle > 0 {
		blockAngleRad := radians(data.BlockAngle)
		angleSteps = int(math.Round(totalArc / blockAngleRad))
		if angleSteps < 1 {
			angleSteps = 1
		}
	}

	if angleSteps > 1 && data.AngleSep > 0 {
		segAngle := totalArc / float64(angleSteps)
		sepAngle := radians(data.AngleSep)
		filledSteps := int(math.Round(previewRatio * float64(angleSteps)))
		if filledSteps > angleSteps {
			filledSteps = angleSteps
		}
		for i := 0; i < angleSteps; i++ {
			segStart := amin + float64(i)*segAngle
			segEnd := segStart + segAngle - sepAngle
			if segEnd <= segStart {
				continue
			}
			if data.RevertValue {
				if i >= angleSteps-filledSteps {
					drawArcFn(segStart, segEnd, barColor)
				} else if emptyColor != nil {
					drawArcFn(segStart, segEnd, emptyColor)
				}
			} else {
				if i < filledSteps {
					drawArcFn(segStart, segEnd, barColor)
				} else if emptyColor != nil {
					drawArcFn(segStart, segEnd, emptyColor)
				}
			}
		}
	} else {
		if data.RevertValue {
			if emptyColor != nil {
				drawArcFn(amin, cur, emptyColor)
			}
			drawArcFn(cur, amax, barColor)
		} else {
			if emptyColor != nil {
				drawArcFn(amin, amax, emptyColor)
			}
			drawArcFn(amin, cur, barColor)
		}
	}

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
	face, err := fc.NewFace(fontPath, fontSize)
	if err != nil {
		return nil
	}
	return face
}

func radians(degrees int) float64 {
	return float64(degrees) * (math.Pi / 180.0)
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	b := img.Bounds()
	x1 := x
	if x1 < b.Min.X {
		x1 = b.Min.X
	}
	y1 := y
	if y1 < b.Min.Y {
		y1 = b.Min.Y
	}
	x2 := x + w
	if x2 > b.Max.X {
		x2 = b.Max.X
	}
	y2 := y + h
	if y2 > b.Max.Y {
		y2 = b.Max.Y
	}
	if x1 >= x2 || y1 >= y2 {
		return
	}
	r32, g32, b32, a32 := c.RGBA()
	px := color.RGBA{R: uint8(r32 >> 8), G: uint8(g32 >> 8), B: uint8(b32 >> 8), A: uint8(a32 >> 8)}
	// fill first row
	for px_ := x1; px_ < x2; px_++ {
		img.SetRGBA(px_, y1, px)
	}
	// copy first row into subsequent rows
	firstOff := img.PixOffset(x1, y1)
	rowLen := (x2 - x1) * 4
	firstRow := img.Pix[firstOff : firstOff+rowLen]
	for py_ := y1 + 1; py_ < y2; py_++ {
		off := img.PixOffset(x1, py_)
		copy(img.Pix[off:off+rowLen], firstRow)
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

func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + t*(float64(b.R)-float64(a.R))),
		G: uint8(float64(a.G) + t*(float64(b.G)-float64(a.G))),
		B: uint8(float64(a.B) + t*(float64(b.B)-float64(a.B))),
		A: uint8(float64(a.A) + t*(float64(b.A)-float64(a.A))),
	}
}

func toRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func fillGradientRGBA(img *image.RGBA, x, y, w, h int, top, bottom color.Color) {
	nt := toRGBA(top)
	nb := toRGBA(bottom)
	b := img.Bounds()
	for py := y; py < y+h; py++ {
		if py < b.Min.Y || py >= b.Max.Y {
			continue
		}
		t := 0.0
		if h > 1 {
			t = float64(py-y) / float64(h-1)
		}
		nc := lerpRGBA(nt, nb, t)
		for px := x; px < x+w; px++ {
			if px < b.Min.X || px >= b.Max.X {
				continue
			}
			img.SetRGBA(px, py, nc)
		}
	}
}

func inRoundCornerRGBA(px, py, x, y, w, h, radius int) bool {
	r := float64(radius)
	dx := float64(px - x)
	dy := float64(py - y)
	rx := float64(x + w - 1 - px)
	ry := float64(y + h - 1 - py)
	if dx < r && dy < r && (r-dx-0.5)*(r-dx-0.5)+(r-dy-0.5)*(r-dy-0.5) > r*r {
		return true
	}
	if rx < r && dy < r && (r-rx-0.5)*(r-rx-0.5)+(r-dy-0.5)*(r-dy-0.5) > r*r {
		return true
	}
	if dx < r && ry < r && (r-dx-0.5)*(r-dx-0.5)+(r-ry-0.5)*(r-ry-0.5) > r*r {
		return true
	}
	if rx < r && ry < r && (r-rx-0.5)*(r-rx-0.5)+(r-ry-0.5)*(r-ry-0.5) > r*r {
		return true
	}
	return false
}

func fillRoundedRGBA(img *image.RGBA, x, y, w, h, radius int, c color.Color) {
	nc := toRGBA(c)
	b := img.Bounds()
	for py := y; py < y+h; py++ {
		if py < b.Min.Y || py >= b.Max.Y {
			continue
		}
		for px := x; px < x+w; px++ {
			if px < b.Min.X || px >= b.Max.X {
				continue
			}
			if !inRoundCornerRGBA(px, py, x, y, w, h, radius) {
				img.SetRGBA(px, py, nc)
			}
		}
	}
}

func fillGradientRoundedRGBA(img *image.RGBA, x, y, w, h, radius int, top, bottom color.Color) {
	nt := toRGBA(top)
	nb := toRGBA(bottom)
	b := img.Bounds()
	for py := y; py < y+h; py++ {
		if py < b.Min.Y || py >= b.Max.Y {
			continue
		}
		t := 0.0
		if h > 1 {
			t = float64(py-y) / float64(h-1)
		}
		nc := lerpRGBA(nt, nb, t)
		for px := x; px < x+w; px++ {
			if px < b.Min.X || px >= b.Max.X {
				continue
			}
			if !inRoundCornerRGBA(px, py, x, y, w, h, radius) {
				img.SetRGBA(px, py, nc)
			}
		}
	}
}

func lerpNRGBAColor(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(float64(a.R) + t*(float64(b.R)-float64(a.R))),
		G: uint8(float64(a.G) + t*(float64(b.G)-float64(a.G))),
		B: uint8(float64(a.B) + t*(float64(b.B)-float64(a.B))),
		A: uint8(float64(a.A) + t*(float64(b.A)-float64(a.A))),
	}
}

func drawArcGradientOnNRGBA(img *image.NRGBA, cx, cy, radius, startAngle, endAngle float64, width int, top, bottom color.Color) {
	nt := colorToNRGBA(top)
	nb := colorToNRGBA(bottom)
	halfW := float64(width) / 2.0
	innerR := radius - halfW
	outerR := radius + halfW
	if innerR < 0 {
		innerR = 0
	}
	minX := int(math.Floor(cx - outerR - 1))
	maxX := int(math.Ceil(cx + outerR + 1))
	minY := int(math.Floor(cy - outerR - 1))
	maxY := int(math.Ceil(cy + outerR + 1))
	bnd := img.Bounds()
	if minX < bnd.Min.X {
		minX = bnd.Min.X
	}
	if minY < bnd.Min.Y {
		minY = bnd.Min.Y
	}
	if maxX > bnd.Max.X {
		maxX = bnd.Max.X
	}
	if maxY > bnd.Max.Y {
		maxY = bnd.Max.Y
	}
	totalH := float64(maxY - minY)
	twoPi := 2 * math.Pi
	for py := minY; py < maxY; py++ {
		for px := minX; px < maxX; px++ {
			dx := float64(px) + 0.5 - cx
			dy := float64(py) + 0.5 - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < innerR || dist > outerR {
				continue
			}
			angle := math.Mod(math.Atan2(dy, dx)+twoPi, twoPi)
			s := math.Mod(startAngle+twoPi, twoPi)
			e := math.Mod(endAngle+twoPi, twoPi)
			inArc := false
			if s <= e {
				if angle >= s && angle <= e {
					inArc = true
				}
			} else {
				if angle >= s || angle <= e {
					inArc = true
				}
			}
			if !inArc {
				continue
			}
			t := 0.0
			if totalH > 0 {
				t = float64(py-minY) / totalH
			}
			img.SetNRGBA(px, py, lerpNRGBAColor(nt, nb, t))
		}
	}
}

func drawCircleOnNRGBA(img *image.NRGBA, cx, cy, radius float64, c color.Color) {
	nc := colorToNRGBA(c)
	rSq := radius * radius
	bnd := img.Bounds()
	for py := int(cy - radius); py <= int(cy+radius); py++ {
		for px := int(cx - radius); px <= int(cx+radius); px++ {
			dx := float64(px) + 0.5 - cx
			dy := float64(py) + 0.5 - cy
			if dx*dx+dy*dy <= rSq {
				if px >= bnd.Min.X && py >= bnd.Min.Y && px < bnd.Max.X && py < bnd.Max.Y {
					img.SetNRGBA(px, py, nc)
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
