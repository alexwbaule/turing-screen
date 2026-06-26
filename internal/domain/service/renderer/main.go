package renderer

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/utils"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type Builder struct {
	log        *logger.Logger
	device     *device.Display
	theme      *theme.Display
	background *image.NRGBA // Composed background (static_images + static_texts)
}

func NewBuilder(l *logger.Logger, v *device.Display, d *theme.Display) *Builder {
	var w, h int
	if d.Width > 0 && d.Height > 0 {
		w, h = d.Width, d.Height
	} else if d.Orientation == theme.PORTRAIT || d.Orientation == theme.REVERSE_PORTRAIT {
		w, h = min(v.Width, v.Height), max(v.Width, v.Height)
	} else {
		w, h = max(v.Width, v.Height), min(v.Width, v.Height)
	}
	return &Builder{
		log:        l,
		device:     v,
		theme:      d,
		background: image.NewNRGBA(image.Rect(0, 0, w, h)),
	}
}

const tolerance = float64(2)

func (b *Builder) BuildBackgroundImage(images map[string]theme.StaticImage) {
	keys := maps.Keys(images)
	slices.Sort(keys)
	for _, name := range keys {
		img := images[name]
		r := image.Rect(img.X, img.Y, img.X+img.BackgroundImage.Bounds().Dx(), img.Y+img.BackgroundImage.Bounds().Dy())
		draw.Draw(b.background, r, img.BackgroundImage, img.BackgroundImage.Bounds().Min, draw.Over)
	}
}

func (b *Builder) GetBackground() *image.NRGBA {
	return b.background
}

func (b *Builder) BuildBackgroundTexts(images map[string]theme.StaticText) {
	keys := maps.Keys(images)
	slices.Sort(keys)
	for _, name := range keys {
		text := images[name]

		// Measure text
		w := measureString(text.Font, text.Text)
		metrics := text.Font.Metrics()
		ascent := metrics.Ascent
		h := fixedToFloat(metrics.Ascent + metrics.Descent)

		// Compute draw origin based on alignment (same semantics as DrawText)
		var drawX float64
		switch text.Align {
		case theme.CENTER:
			drawX = float64(text.X) - w/2
		case theme.RIGHT:
			drawX = float64(text.X) - w
		default: // LEFT
			drawX = float64(text.X)
		}

		bgX := drawX - tolerance
		bgY := float64(text.Y)

		if text.BackgroundColor != nil && text.BackgroundColor != color.Transparent {
			fillRect(b.background, int(bgX), int(bgY), int(w+tolerance+tolerance), int(h), text.BackgroundColor)
		}

		dot := fixed.Point26_6{
			X: fixed.Int26_6(int((drawX - (tolerance / 2)) * 64)),
			Y: fixed.I(int(bgY-(tolerance/2))) + ascent,
		}

		d := &font.Drawer{
			Dst:  b.background,
			Src:  image.NewUniform(text.FontColor),
			Face: text.Font,
			Dot:  dot,
		}
		d.DrawString(text.Text)
	}
}

func (b *Builder) DrawText(text string, stat *theme.Text, defaultSize int) (image.Image, int, int) {

	numb := imageToNRGBA(b.background)

	d := &font.Drawer{
		Dst:  numb,
		Src:  image.NewUniform(stat.FontColor),
		Face: stat.Font,
	}

	// Measure crop width: use the larger of placeholder or actual text
	var cropWidth int
	if stat.Placeholder != "" {
		placeholderWidth := d.MeasureString(stat.Placeholder).Ceil()
		textWidth := d.MeasureString(text).Ceil()
		if placeholderWidth > textWidth {
			cropWidth = placeholderWidth
		} else {
			cropWidth = textWidth
		}
	} else {
		cropWidth = d.MeasureString(text).Ceil()
	}

	metrics := stat.Font.Metrics()
	cropHeight := (metrics.Ascent + metrics.Descent).Ceil()

	// Override with layout Width/Height if defined
	if stat.Width > 0 {
		cropWidth = stat.Width
	}
	if stat.Height > 0 {
		cropHeight = stat.Height
	}

	// Measure actual text width using fixed-point for sub-pixel precision
	textWidth := d.MeasureString(text)
	cropWidthFixed := fixed.I(cropWidth)

	// Calculate crop origin and text position based on alignment
	var dotX fixed.Int26_6
	cropX := stat.X // left edge of crop region

	switch stat.Align {
	case theme.CENTER:
		// X is the CENTER point — crop starts at X - cropWidth/2
		cropX = stat.X - cropWidth/2
		if cropX < 0 {
			cropX = 0
		}
		dotX = fixed.I(cropX) + (cropWidthFixed-textWidth)/2
	case theme.RIGHT:
		// X is the RIGHT edge — crop starts at X - cropWidth
		cropX = stat.X - cropWidth
		if cropX < 0 {
			cropX = 0
		}
		dotX = fixed.I(cropX) + cropWidthFixed - textWidth
	default: // LEFT
		dotX = fixed.I(stat.X)
	}

	// Y position: top of crop + ascent = baseline
	dotY := fixed.I(stat.Y) + metrics.Ascent

	// Draw
	d.Dot = fixed.Point26_6{X: dotX, Y: dotY}
	d.DrawString(text)

	// Crop the region
	crp := image.Rect(cropX, stat.Y, cropX+cropWidth, stat.Y+cropHeight)
	dst := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))
	drawCrop(dst, numb, crp)

	return dst, cropX, stat.Y
}

func (b *Builder) DrawProgressBar(value float64, stat *theme.Graph) image.Image {

	numb := imageToNRGBA(b.background)

	ratio := value / float64(stat.MaxValue-stat.MinValue)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	if stat.RevertValue {
		ratio = 1 - ratio
	}

	direction := strings.ToLower(stat.Direction)
	if direction == "" {
		direction = "left"
	}

	var filledSize int
	switch direction {
	case "left", "right":
		filledSize = int(math.Round(ratio * float64(stat.Width)))
	case "up", "down":
		filledSize = int(math.Round(ratio * float64(stat.Height)))
	}

	if stat.BackgroundStyle.BackgroundColor != nil {
		_, _, _, a := stat.BackgroundStyle.BackgroundColor.RGBA()
		if a > 0 {
			fillRect(numb, stat.X, stat.Y, stat.Width, stat.Height, stat.BackgroundStyle.BackgroundColor)
		}
	}

	hasGradient := stat.GradientColor != nil
	hasRound := stat.CornerRadius > 0

	fillBar := func(x, y, w, h int, clr color.Color) {
		if w <= 0 || h <= 0 || clr == nil {
			return
		}
		if hasGradient && hasRound {
			fillGradientRoundedRect(numb, x, y, w, h, stat.CornerRadius, stat.GradientColor, clr)
		} else if hasGradient {
			fillGradientRect(numb, x, y, w, h, stat.GradientColor, clr)
		} else if hasRound {
			fillRoundedRect(numb, x, y, w, h, stat.CornerRadius, clr)
		} else {
			fillRect(numb, x, y, w, h, clr)
		}
	}

	// BlockWidth takes priority over Steps
	steps := stat.Steps
	if stat.BlockWidth > 0 {
		gap := stat.StepGap
		if gap < 0 {
			gap = 0
		}
		steps = stat.Width / (stat.BlockWidth + gap)
		if steps < 1 {
			steps = 1
		}
	}

	if steps > 0 && stat.StepGap > 0 {
		b.drawSegmentedBar(numb, stat, direction, filledSize, steps, fillBar)
	} else {
		switch direction {
		case "left":
			fillBar(stat.X, stat.Y, filledSize, stat.Height, stat.BarColor)
		case "right":
			fillBar(stat.X+stat.Width-filledSize, stat.Y, filledSize, stat.Height, stat.BarColor)
		case "up":
			fillBar(stat.X, stat.Y+stat.Height-filledSize, stat.Width, filledSize, stat.BarColor)
		case "down":
			fillBar(stat.X, stat.Y, stat.Width, filledSize, stat.BarColor)
		}
	}

	if stat.BorderWidth > 0 {
		strokeBorderRect(numb, stat.X, stat.Y, stat.Width, stat.Height, stat.BorderWidth, stat.BarColor)
	} else if stat.BarOutline {
		strokeRect(numb, stat.X, stat.Y, stat.Width, stat.Height, stat.BarColor)
	}

	crp := image.Rect(stat.X, stat.Y, stat.X+stat.Width, stat.Y+stat.Height)
	dst := image.NewRGBA(image.Rect(0, 0, stat.Width, stat.Height))
	drawCrop(dst, numb, crp)

	return dst
}

func (b *Builder) drawSegmentedBar(numb *image.NRGBA, stat *theme.Graph, direction string, filledSize, steps int, fillBar func(x, y, w, h int, clr color.Color)) {
	gap := stat.StepGap

	switch direction {
	case "left", "right":
		segW := stat.BlockWidth
		if segW <= 0 {
			segW = (stat.Width - (steps-1)*gap) / steps
		}
		if segW < 1 {
			segW = 1
		}
		filledSteps := int(math.Round(float64(filledSize) / float64(stat.Width) * float64(steps)))
		for i := 0; i < steps; i++ {
			var sx int
			if direction == "left" {
				sx = stat.X + i*(segW+gap)
			} else {
				sx = stat.X + stat.Width - (i+1)*(segW+gap) + gap
			}
			if i < filledSteps {
				fillBar(sx, stat.Y, segW, stat.Height, stat.BarColor)
			}
		}
	case "up", "down":
		segH := stat.BlockWidth
		if segH <= 0 {
			segH = (stat.Height - (steps-1)*gap) / steps
		}
		if segH < 1 {
			segH = 1
		}
		filledSteps := int(math.Round(float64(filledSize) / float64(stat.Height) * float64(steps)))
		for i := 0; i < steps; i++ {
			var sy int
			if direction == "up" {
				sy = stat.Y + stat.Height - (i+1)*(segH+gap) + gap
			} else {
				sy = stat.Y + i*(segH+gap)
			}
			if i < filledSteps {
				fillBar(stat.X, sy, stat.Width, segH, stat.BarColor)
			}
		}
	}
}

func (b *Builder) DrawRadialProgressBar(value float64, stat *theme.Radial) image.Image {

	numb := imageToNRGBA(b.background)

	diameter := 2 * stat.Radius
	x, y := float64(stat.X), float64(stat.Y)

	ratio := value / float64(stat.MaxValue-stat.MinValue)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}

	amin := utils.Radians(stat.AngleStart)
	amax := utils.Radians(180 + stat.AngleStart + stat.AngleEnd)
	totalArc := amax - amin
	filledAngle := totalArc * ratio

	var cur float64
	if stat.RevertValue {
		cur = amax - filledAngle
	} else {
		cur = amin + filledAngle
	}
	if cur > amax {
		cur = amax
	}

	b.log.Debugf("A: %f, C: %f %f", amin, amax, cur)

	if stat.ShowText {
		measure := fmt.Sprintf("%3.f", value)
		if stat.ShowUnit {
			measure = fmt.Sprintf("%3.f%%", value)
		}

		w := measureString(stat.Font, measure)
		ascent := fixedAscent(stat.Font)
		descent := fixedDescent(stat.Font)
		totalH := ascent + descent

		drawX := fixed.Int26_6(int((x - w/2) * 64))
		drawY := fixed.I(int(y)) + totalH/2 - descent + fixed.I(1)

		d := &font.Drawer{
			Dst:  numb,
			Src:  image.NewUniform(stat.FontColor),
			Face: stat.Font,
			Dot:  fixed.Point26_6{X: drawX, Y: drawY},
		}
		d.DrawString(measure)
	}

	arcRadius := float64(stat.Radius - (stat.Width / 2))

	barClr := stat.BarColor
	var emptyClr color.Color
	if bgClr := stat.BackgroundStyle.BackgroundColor; bgClr != nil {
		if _, _, _, a := bgClr.RGBA(); a > 0 {
			emptyClr = bgClr
		}
	}
	if stat.Revert {
		barClr, emptyClr = emptyClr, barClr
	}

	hasGradient := stat.GradientColor != nil

	drawArcFn := func(a1, a2 float64, clr color.Color) {
		if clr == nil || a2 <= a1 {
			return
		}
		if hasGradient {
			drawArcGradient(numb, x, y, arcRadius, a1, a2, stat.Width, stat.GradientColor, clr)
		} else {
			drawArc(numb, x, y, arcRadius, a1, a2, stat.Width, clr)
		}
		if stat.Round {
			r := float64(stat.Width) / 2.0
			sx := x + arcRadius*math.Cos(a1)
			sy := y + arcRadius*math.Sin(a1)
			ex := x + arcRadius*math.Cos(a2)
			ey := y + arcRadius*math.Sin(a2)
			drawFilledCircle(numb, sx, sy, r, clr)
			drawFilledCircle(numb, ex, ey, r, clr)
		}
	}

	// Determine effective step count (BlockAngle takes priority)
	angleSteps := stat.AngleSteps
	if stat.BlockAngle > 0 {
		blockAngleRad := utils.Radians(stat.BlockAngle)
		angleSteps = int(math.Round(totalArc / blockAngleRad))
		if angleSteps < 1 {
			angleSteps = 1
		}
	}

	if angleSteps > 1 && stat.AngleSep > 0 {
		segAngle := totalArc / float64(angleSteps)
		sepAngle := utils.Radians(stat.AngleSep)
		filledSteps := int(math.Round(ratio * float64(angleSteps)))
		if filledSteps > angleSteps {
			filledSteps = angleSteps
		}
		for i := 0; i < angleSteps; i++ {
			segStart := amin + float64(i)*segAngle
			segEnd := segStart + segAngle - sepAngle
			if segEnd <= segStart {
				continue
			}
			if stat.RevertValue {
				if i >= angleSteps-filledSteps {
					drawArcFn(segStart, segEnd, barClr)
				} else if emptyClr != nil {
					drawArcFn(segStart, segEnd, emptyClr)
				}
			} else {
				if i < filledSteps {
					drawArcFn(segStart, segEnd, barClr)
				} else if emptyClr != nil {
					drawArcFn(segStart, segEnd, emptyClr)
				}
			}
		}
	} else {
		if stat.RevertValue {
			if emptyClr != nil {
				drawArcFn(amin, cur, emptyClr)
			}
			drawArcFn(cur, amax, barClr)
		} else {
			if emptyClr != nil {
				drawArcFn(amin, amax, emptyClr)
			}
			drawArcFn(amin, cur, barClr)
		}
	}

	crp := image.Rect(stat.X-stat.Radius, stat.Y-stat.Radius, stat.X+stat.Radius, stat.Y+stat.Radius)
	dst := image.NewRGBA(image.Rect(0, 0, diameter, diameter))
	drawCrop(dst, numb, crp)

	return dst
}

// DrawStatusBar draws a progress bar with a circular indicator at the current value position.
func (b *Builder) DrawStatusBar(value float64, stat *theme.StatusBar) image.Image {
	numb := imageToNRGBA(b.background)

	barHeight := stat.Height
	barY := stat.Y + (stat.Height-barHeight)/2

	// Calculate filled position
	ratio := value / float64(stat.MaxValue-stat.MinValue)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	filledWidth := int(math.Round(ratio * float64(stat.Width)))

	// Draw bar background (optional)
	// Draw filled portion
	fillRect(numb, stat.X, barY, filledWidth, barHeight, stat.BarColor)

	// Draw indicator circle at the position
	indicatorR := stat.IndicatorRadius
	if indicatorR <= 0 {
		indicatorR = barHeight // default: same as bar height
	}
	cx := float64(stat.X + filledWidth)
	cy := float64(stat.Y + stat.Height/2)
	drawFilledCircle(numb, cx, cy, float64(indicatorR), stat.IndicatorColor)

	// Crop region (account for indicator extending beyond bar)
	cropX := stat.X - indicatorR
	cropY := stat.Y - indicatorR
	cropW := stat.Width + indicatorR*2
	cropH := stat.Height + indicatorR*2
	if cropX < 0 {
		cropX = 0
	}
	if cropY < 0 {
		cropY = 0
	}

	crp := image.Rect(cropX, cropY, cropX+cropW, cropY+cropH)
	dst := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	drawCrop(dst, numb, crp)

	return dst
}

// DrawGauge draws a needle/pointer at an angle proportional to the value.
func (b *Builder) DrawGauge(value float64, stat *theme.Gauge) image.Image {
	numb := imageToNRGBA(b.background)

	diameter := 2 * stat.Radius
	x, y := float64(stat.X), float64(stat.Y)

	// Calculate needle angle
	ratio := value / float64(stat.MaxValue-stat.MinValue)
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	totalAngle := float64(180 + stat.AngleStart + stat.AngleEnd)
	needleAngle := utils.Radians(stat.AngleStart) + ratio*utils.Radians(int(totalAngle))

	// Draw needle (line from center to edge)
	needleLen := float64(stat.Radius) * 0.85
	needleWidth := stat.NeedleWidth
	if needleWidth <= 0 {
		needleWidth = 2
	}

	endX := x + needleLen*math.Cos(needleAngle)
	endY := y + needleLen*math.Sin(needleAngle)
	drawLine(numb, x, y, endX, endY, needleWidth, stat.NeedleColor)

	// Draw center dot
	drawFilledCircle(numb, x, y, float64(needleWidth+1), stat.NeedleColor)

	// Draw text in center if ShowText
	if stat.ShowText {
		fontSize := stat.Radius / 3
		if fontSize <= 0 {
			fontSize = 10
		}
		measure := fmt.Sprintf("%3.f", value)
		if stat.ShowUnit {
			measure = fmt.Sprintf("%3.f%%", value)
		}

		w := measureString(stat.Font, measure)
		ascent := fixedAscent(stat.Font)
		descent := fixedDescent(stat.Font)
		totalH := ascent + descent

		drawDotX := fixed.Int26_6(int((x - w/2) * 64))
		drawDotY := fixed.I(int(y)) + totalH/2 - descent + fixed.I(int(float64(stat.Radius)*0.35))

		d := &font.Drawer{
			Dst:  numb,
			Src:  image.NewUniform(stat.FontColor),
			Face: stat.Font,
			Dot:  fixed.Point26_6{X: drawDotX, Y: drawDotY},
		}
		d.DrawString(measure)
	}

	// Crop
	crp := image.Rect(stat.X-stat.Radius, stat.Y-stat.Radius, stat.X+stat.Radius, stat.Y+stat.Radius)
	dst := image.NewRGBA(image.Rect(0, 0, diameter, diameter))
	drawCrop(dst, numb, crp)

	return dst
}

// DrawChart draws a time-series chart from the Chart's ring buffer.
// Style "line" draws connected line segments; "bar" (default) draws vertical bars.
func (b *Builder) DrawChart(stat *theme.Chart) image.Image {
	numb := imageToNRGBA(b.background)

	samples := stat.GetSamples()
	colStep := stat.ColumnWidth + stat.ColumnGap
	if colStep <= 0 {
		colStep = 2
	}

	valueRange := float64(stat.MaxValue - stat.MinValue)
	if valueRange <= 0 {
		valueRange = 100
	}

	// Draw border
	if stat.BorderWidth > 0 {
		strokeRect(numb, stat.X, stat.Y, stat.Width, stat.Height, stat.LineColor)
	}

	style := strings.ToLower(stat.Style)

	if style == "line" {
		// Line chart: connect points with lines, fill below
		if len(samples) >= 2 {
			// Calculate X spacing to fill the chart width
			spacing := float64(stat.Width) / float64(len(samples)-1)
			if spacing < 1 {
				spacing = 1
			}

			// Draw filled area below the line
			for i := 0; i < len(samples)-1; i++ {
				r1 := samples[i] / valueRange
				r2 := samples[i+1] / valueRange
				if r1 > 1 {
					r1 = 1
				}
				if r2 > 1 {
					r2 = 1
				}
				if r1 < 0 {
					r1 = 0
				}
				if r2 < 0 {
					r2 = 0
				}

				x1 := float64(stat.X) + float64(i)*spacing
				x2 := float64(stat.X) + float64(i+1)*spacing
				y1 := float64(stat.Y+stat.Height) - r1*float64(stat.Height)
				y2 := float64(stat.Y+stat.Height) - r2*float64(stat.Height)

				// Fill below the line segment
				for px := int(x1); px < int(x2) && px < stat.X+stat.Width; px++ {
					t := (float64(px) - x1) / (x2 - x1)
					lineY := y1 + t*(y2-y1)
					for py := int(lineY); py < stat.Y+stat.Height; py++ {
						if px >= stat.X && py >= stat.Y {
							numb.Set(px, py, stat.FillColor)
						}
					}
				}

				// Draw line segment on top
				drawLine(numb, x1, y1, x2, y2, 1, stat.LineColor)
			}
		}
	} else {
		// Bar chart (default): draw columns right-to-left (newest on right)
		for i := len(samples) - 1; i >= 0; i-- {
			colIndex := len(samples) - 1 - i
			colX := stat.X + stat.Width - (colIndex+1)*colStep
			if colX < stat.X {
				break
			}

			ratio := samples[i] / valueRange
			if ratio > 1 {
				ratio = 1
			}
			if ratio < 0 {
				ratio = 0
			}
			barH := int(math.Round(ratio * float64(stat.Height)))
			barY := stat.Y + stat.Height - barH

			fillRect(numb, colX, barY, stat.ColumnWidth, barH, stat.FillColor)
		}
	}

	// Crop
	crp := image.Rect(stat.X, stat.Y, stat.X+stat.Width, stat.Y+stat.Height)
	dst := image.NewRGBA(image.Rect(0, 0, stat.Width, stat.Height))
	drawCrop(dst, numb, crp)

	return dst
}

// =====================================================================
// Helper functions (pure Go, no CGO)
// =====================================================================

func toRendererNRGBA(c color.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func lerpRendererNRGBA(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(float64(a.R) + t*(float64(b.R)-float64(a.R))),
		G: uint8(float64(a.G) + t*(float64(b.G)-float64(a.G))),
		B: uint8(float64(a.B) + t*(float64(b.B)-float64(a.B))),
		A: uint8(float64(a.A) + t*(float64(b.A)-float64(a.A))),
	}
}

func fillGradientRect(dst *image.NRGBA, x, y, w, h int, top, bottom color.Color) {
	nt := toRendererNRGBA(top)
	nb := toRendererNRGBA(bottom)
	bounds := dst.Bounds()
	for py := y; py < y+h; py++ {
		if py < bounds.Min.Y || py >= bounds.Max.Y {
			continue
		}
		t := 0.0
		if h > 1 {
			t = float64(py-y) / float64(h-1)
		}
		nc := lerpRendererNRGBA(nt, nb, t)
		for px := x; px < x+w; px++ {
			if px < bounds.Min.X || px >= bounds.Max.X {
				continue
			}
			dst.SetNRGBA(px, py, nc)
		}
	}
}

func inRoundedCornerR(px, py, x, y, w, h, radius int) bool {
	r := float64(radius)
	dx := float64(px - x)
	dy := float64(py - y)
	rx := float64(x + w - 1 - px)
	ry := float64(y + h - 1 - py)
	if dx < r && dy < r {
		if (r-dx-0.5)*(r-dx-0.5)+(r-dy-0.5)*(r-dy-0.5) > r*r {
			return true
		}
	}
	if rx < r && dy < r {
		if (r-rx-0.5)*(r-rx-0.5)+(r-dy-0.5)*(r-dy-0.5) > r*r {
			return true
		}
	}
	if dx < r && ry < r {
		if (r-dx-0.5)*(r-dx-0.5)+(r-ry-0.5)*(r-ry-0.5) > r*r {
			return true
		}
	}
	if rx < r && ry < r {
		if (r-rx-0.5)*(r-rx-0.5)+(r-ry-0.5)*(r-ry-0.5) > r*r {
			return true
		}
	}
	return false
}

func fillRoundedRect(dst *image.NRGBA, x, y, w, h, radius int, c color.Color) {
	nc := toRendererNRGBA(c)
	bounds := dst.Bounds()
	for py := y; py < y+h; py++ {
		if py < bounds.Min.Y || py >= bounds.Max.Y {
			continue
		}
		for px := x; px < x+w; px++ {
			if px < bounds.Min.X || px >= bounds.Max.X {
				continue
			}
			if !inRoundedCornerR(px, py, x, y, w, h, radius) {
				dst.SetNRGBA(px, py, nc)
			}
		}
	}
}

func fillGradientRoundedRect(dst *image.NRGBA, x, y, w, h, radius int, top, bottom color.Color) {
	nt := toRendererNRGBA(top)
	nb := toRendererNRGBA(bottom)
	bounds := dst.Bounds()
	for py := y; py < y+h; py++ {
		if py < bounds.Min.Y || py >= bounds.Max.Y {
			continue
		}
		t := 0.0
		if h > 1 {
			t = float64(py-y) / float64(h-1)
		}
		nc := lerpRendererNRGBA(nt, nb, t)
		for px := x; px < x+w; px++ {
			if px < bounds.Min.X || px >= bounds.Max.X {
				continue
			}
			if !inRoundedCornerR(px, py, x, y, w, h, radius) {
				dst.SetNRGBA(px, py, nc)
			}
		}
	}
}

func strokeBorderRect(dst *image.NRGBA, x, y, w, h, bw int, c color.Color) {
	for i := 0; i < bw; i++ {
		strokeRect(dst, x+i, y+i, w-2*i, h-2*i, c)
	}
}

func drawArcGradient(dst *image.NRGBA, cx, cy, radius, startAngle, endAngle float64, width int, top, bottom color.Color) {
	nt := toRendererNRGBA(top)
	nb := toRendererNRGBA(bottom)
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

	bounds := dst.Bounds()
	if minX < bounds.Min.X {
		minX = bounds.Min.X
	}
	if minY < bounds.Min.Y {
		minY = bounds.Min.Y
	}
	if maxX > bounds.Max.X {
		maxX = bounds.Max.X
	}
	if maxY > bounds.Max.Y {
		maxY = bounds.Max.Y
	}

	totalH := float64(maxY - minY)
	for py := minY; py < maxY; py++ {
		for px := minX; px < maxX; px++ {
			dx := float64(px) + 0.5 - cx
			dy := float64(py) + 0.5 - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < innerR || dist > outerR {
				continue
			}
			angle := math.Atan2(dy, dx)
			if angle < 0 {
				angle += 2 * math.Pi
			}
			if !isAngleInRange(angle, startAngle, endAngle) {
				continue
			}
			t := 0.0
			if totalH > 0 {
				t = float64(py-minY) / totalH
			}
			dst.SetNRGBA(px, py, lerpRendererNRGBA(nt, nb, t))
		}
	}
}

// imageToNRGBA converts any image.Image to *image.NRGBA (always returns a new copy).
func imageToNRGBA(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	dst := image.NewNRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}

// fillRect fills a solid rectangle on dst.
func fillRect(dst *image.NRGBA, x, y, w, h int, c color.Color) {
	r, g, b, a := c.RGBA()
	nc := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			if px >= 0 && py >= 0 && px < dst.Bounds().Max.X && py < dst.Bounds().Max.Y {
				dst.SetNRGBA(px, py, nc)
			}
		}
	}
}

// strokeRect draws a 1px outline rectangle on dst.
func strokeRect(dst *image.NRGBA, x, y, w, h int, c color.Color) {
	r, g, b, a := c.RGBA()
	nc := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	// Top & bottom
	for px := x; px < x+w; px++ {
		setPixelSafe(dst, px, y, nc)
		setPixelSafe(dst, px, y+h-1, nc)
	}
	// Left & right
	for py := y; py < y+h; py++ {
		setPixelSafe(dst, x, py, nc)
		setPixelSafe(dst, x+w-1, py, nc)
	}
}

// setPixelSafe sets a pixel with bounds checking.
func setPixelSafe(dst *image.NRGBA, x, y int, c color.NRGBA) {
	if x >= 0 && y >= 0 && x < dst.Bounds().Max.X && y < dst.Bounds().Max.Y {
		dst.SetNRGBA(x, y, c)
	}
}

// drawArc draws a thick arc on dst using Bresenham-style pixel plotting.
// cx, cy = center; radius = arc radius; startAngle, endAngle in radians;
// width = line thickness; c = color.
func drawArc(dst *image.NRGBA, cx, cy, radius, startAngle, endAngle float64, width int, c color.Color) {
	r, g, b, a := c.RGBA()
	nc := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}

	// Normalize angles
	for startAngle < 0 {
		startAngle += 2 * math.Pi
	}
	for endAngle < 0 {
		endAngle += 2 * math.Pi
	}

	halfWidth := float64(width) / 2.0
	innerR := radius - halfWidth
	outerR := radius + halfWidth

	if innerR < 0 {
		innerR = 0
	}

	// Determine bounding box for scanning
	minX := int(math.Floor(cx - outerR - 1))
	maxX := int(math.Ceil(cx + outerR + 1))
	minY := int(math.Floor(cy - outerR - 1))
	maxY := int(math.Ceil(cy + outerR + 1))

	bounds := dst.Bounds()
	if minX < bounds.Min.X {
		minX = bounds.Min.X
	}
	if minY < bounds.Min.Y {
		minY = bounds.Min.Y
	}
	if maxX > bounds.Max.X {
		maxX = bounds.Max.X
	}
	if maxY > bounds.Max.Y {
		maxY = bounds.Max.Y
	}

	for py := minY; py < maxY; py++ {
		for px := minX; px < maxX; px++ {
			dx := float64(px) + 0.5 - cx
			dy := float64(py) + 0.5 - cy
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist < innerR || dist > outerR {
				continue
			}

			// Check angle
			angle := math.Atan2(dy, dx)
			if angle < 0 {
				angle += 2 * math.Pi
			}

			if isAngleInRange(angle, startAngle, endAngle) {
				dst.SetNRGBA(px, py, nc)
			}
		}
	}
}

// isAngleInRange checks if angle is between start and end (handling wrap-around).
func isAngleInRange(angle, start, end float64) bool {
	// Normalize all to [0, 2*Pi)
	twoPi := 2 * math.Pi
	angle = math.Mod(angle, twoPi)
	if angle < 0 {
		angle += twoPi
	}
	start = math.Mod(start, twoPi)
	if start < 0 {
		start += twoPi
	}
	end = math.Mod(end, twoPi)
	if end < 0 {
		end += twoPi
	}

	if start <= end {
		return angle >= start && angle <= end
	}
	// Wraps around 0
	return angle >= start || angle <= end
}

// drawCrop copies the region `crp` from src into dst (which should be sized to crp dimensions).
func drawCrop(dst *image.RGBA, src image.Image, crp image.Rectangle) {
	draw.Draw(dst, dst.Bounds(), src, crp.Min, draw.Src)
}

// drawFilledCircle draws a solid circle on dst.
func drawFilledCircle(dst *image.NRGBA, cx, cy, radius float64, c color.Color) {
	r, g, b, a := c.RGBA()
	nc := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	rSq := radius * radius
	for py := int(cy - radius); py <= int(cy+radius); py++ {
		for px := int(cx - radius); px <= int(cx+radius); px++ {
			dx := float64(px) + 0.5 - cx
			dy := float64(py) + 0.5 - cy
			if dx*dx+dy*dy <= rSq {
				if px >= 0 && py >= 0 && px < dst.Bounds().Dx() && py < dst.Bounds().Dy() {
					dst.SetNRGBA(px, py, nc)
				}
			}
		}
	}
}

// drawLine draws a thick line from (x1,y1) to (x2,y2) on dst.
func drawLine(dst *image.NRGBA, x1, y1, x2, y2 float64, width int, c color.Color) {
	r, g, b, a := c.RGBA()
	nc := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}

	dx := x2 - x1
	dy := y2 - y1
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return
	}

	halfW := float64(width) / 2.0
	steps := int(length * 2)
	if steps < 10 {
		steps = 10
	}

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		px := x1 + dx*t
		py := y1 + dy*t

		for wy := -halfW; wy <= halfW; wy++ {
			for wx := -halfW; wx <= halfW; wx++ {
				if wx*wx+wy*wy <= halfW*halfW {
					ix := int(math.Round(px + wx))
					iy := int(math.Round(py + wy))
					if ix >= 0 && iy >= 0 && ix < dst.Bounds().Dx() && iy < dst.Bounds().Dy() {
						dst.SetNRGBA(ix, iy, nc)
					}
				}
			}
		}
	}
}

// measureString returns the width of the string in float64 pixels.
func measureString(face font.Face, s string) float64 {
	advance := font.MeasureString(face, s)
	return fixedToFloat(advance)
}

// fixedAscent returns the ascent as fixed.Int26_6.
func fixedAscent(face font.Face) fixed.Int26_6 {
	return face.Metrics().Ascent
}

// fixedDescent returns the descent as fixed.Int26_6.
func fixedDescent(face font.Face) fixed.Int26_6 {
	return face.Metrics().Descent
}

// fixedToFloat converts fixed.Int26_6 to float64.
func fixedToFloat(v fixed.Int26_6) float64 {
	return float64(v) / 64.0
}
