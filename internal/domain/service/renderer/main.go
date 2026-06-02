package renderer

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strings"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type Builder struct {
	log    *logger.Logger
	device *device.Display
	theme  *theme.Display
}

func NewBuilder(l *logger.Logger, v *device.Display, d *theme.Display) *Builder {
	return &Builder{
		log:    l,
		device: v,
		theme:  d,
	}
}

const tolerance = float64(2)

func (b *Builder) BuildBackgroundImage(images map[string]theme.StaticImage) image.Image {
	var numb *image.NRGBA

	if b.theme.Orientation == theme.PORTRAIT || b.theme.Orientation == theme.REVERSE_PORTRAIT {
		numb = image.NewNRGBA(image.Rect(0, 0, b.device.Height, b.device.Width))
	} else {
		numb = image.NewNRGBA(image.Rect(0, 0, b.device.Width, b.device.Height))
	}

	keys := maps.Keys(images)
	slices.Sort(keys)
	for _, name := range keys {
		img := images[name]
		b.log.Debugf("Build Background Image [%#v]", img)
		r := image.Rect(img.X, img.Y, img.X+img.BackgroundImage.Bounds().Dx(), img.Y+img.BackgroundImage.Bounds().Dy())
		draw.Draw(numb, r, img.BackgroundImage, img.BackgroundImage.Bounds().Min, draw.Over)
	}

	return numb
}

func (b *Builder) BuildBackgroundTexts(background image.Image, images map[string]theme.StaticText) image.Image {
	dst := imageToNRGBA(background)

	keys := maps.Keys(images)
	slices.Sort(keys)
	for _, name := range keys {
		text := images[name]

		// Measure text
		w := measureString(text.Font, text.Text)
		metrics := text.Font.Metrics()
		ascent := metrics.Ascent
		h := fixedToFloat(metrics.Ascent + metrics.Descent)

		x := float64(text.X) - tolerance
		y := float64(text.Y)

		if text.BackgroundColor != nil && text.BackgroundColor != color.Transparent {
			fillRect(dst, int(x), int(y), int(w+tolerance+tolerance), int(h), text.BackgroundColor)
		}

		dot := fixed.Point26_6{
			X: fixed.Int26_6(int((float64(text.X) - (tolerance / 2)) * 64)),
			Y: fixed.I(int(float64(text.Y)-(tolerance/2))) + ascent,
		}

		d := &font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(text.FontColor),
			Face: text.Font,
			Dot:  dot,
		}
		d.DrawString(text.Text)
	}

	return dst
}

func (b *Builder) DrawText(text string, stat *theme.Text, defaultSize int) image.Image {
	var numb *image.NRGBA

	if stat.BackgroundImage == nil {
		if b.theme.Orientation == theme.PORTRAIT || b.theme.Orientation == theme.REVERSE_PORTRAIT {
			numb = imageToNRGBA(utils.CreateImage(b.device.Height, b.device.Width, stat.BackgroundColor))
		} else {
			numb = imageToNRGBA(utils.CreateImage(b.device.Width, b.device.Height, stat.BackgroundColor))
		}
	} else {
		numb = imageToNRGBA(stat.BackgroundImage)
	}

	// Determine crop size using a reference measure string
	// Priority: theme SIZE > sensor default > text length
	charCount := utils.CountStr(text)
	if stat.Size > 0 {
		charCount = stat.Size
	} else if defaultSize > charCount {
		charCount = defaultSize
	}

	d := &font.Drawer{
		Dst:  numb,
		Src:  image.NewUniform(stat.FontColor),
		Face: stat.Font,
	}

	// Measure crop dimensions using reference string of "8"s
	measure := strings.Repeat("8", charCount)
	cropWidth := d.MeasureString(measure).Ceil()
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

	// Calculate X position within crop region based on alignment
	var dotX fixed.Int26_6
	switch stat.Align {
	case theme.CENTER:
		dotX = fixed.I(stat.X) + (cropWidthFixed-textWidth)/2
	case theme.RIGHT:
		dotX = fixed.I(stat.X) + cropWidthFixed - textWidth
	default: // LEFT
		dotX = fixed.I(stat.X)
	}

	// Y position: top of crop + ascent = baseline
	dotY := fixed.I(stat.Y) + metrics.Ascent

	// Draw
	d.Dot = fixed.Point26_6{X: dotX, Y: dotY}
	d.DrawString(text)

	// Crop the region
	crp := image.Rect(stat.X, stat.Y, stat.X+cropWidth, stat.Y+cropHeight)
	dst := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))
	drawCrop(dst, numb, crp)

	return dst
}

func (b *Builder) DrawProgressBar(value float64, stat *theme.Graph) image.Image {
	var numb *image.NRGBA

	if stat.BackgroundImage == nil {
		if b.theme.Orientation == theme.PORTRAIT || b.theme.Orientation == theme.REVERSE_PORTRAIT {
			numb = imageToNRGBA(utils.CreateImage(b.device.Height, b.device.Width, color.Transparent))
		} else {
			numb = imageToNRGBA(utils.CreateImage(b.device.Width, b.device.Height, color.Transparent))
		}
	} else {
		numb = imageToNRGBA(stat.BackgroundImage)
	}

	barFilledWidth := int(math.Round(value / float64(stat.MaxValue-stat.MinValue) * float64(stat.Width)))

	// Fill the bar
	fillRect(numb, stat.X, stat.Y, barFilledWidth, stat.Height, stat.BarColor)

	// Draw outline if requested
	if stat.BarOutline {
		strokeRect(numb, stat.X, stat.Y, stat.Width, stat.Height, stat.BarColor)
	}

	// Crop the region
	crp := image.Rect(stat.X, stat.Y, stat.X+stat.Width, stat.Y+stat.Height)
	dst := image.NewRGBA(image.Rect(0, 0, stat.Width, stat.Height))
	drawCrop(dst, numb, crp)

	return dst
}

func (b *Builder) DrawRadialProgressBar(value float64, stat *theme.Radial) image.Image {
	var numb *image.NRGBA

	if stat.BackgroundImage == nil {
		if b.theme.Orientation == theme.PORTRAIT || b.theme.Orientation == theme.REVERSE_PORTRAIT {
			numb = imageToNRGBA(utils.CreateImage(b.device.Height, b.device.Width, stat.BackgroundColor))
		} else {
			numb = imageToNRGBA(utils.CreateImage(b.device.Width, b.device.Height, stat.BackgroundColor))
		}
	} else {
		numb = imageToNRGBA(stat.BackgroundImage)
	}

	diameter := 2 * stat.Radius
	x, y := float64(stat.X), float64(stat.Y)

	amin := utils.Radians(stat.AngleStart)
	amax := utils.Radians(180 + stat.AngleStart + stat.AngleEnd)

	total := (value * float64(180+stat.AngleStart+stat.AngleEnd)) / 100
	cur := utils.Radians(int(total) + stat.AngleStart)

	if cur > amax {
		cur = amax
	}

	b.log.Debugf("A: %f, C: %f %f", amin, amax, cur)

	// Draw text in center if ShowText
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

	// Draw arc
	arcRadius := float64(stat.Radius - (stat.Width / 2))
	drawArc(numb, x, y, arcRadius, amin, cur, stat.Width, stat.BarColor)

	// Crop the region
	crp := image.Rect(stat.X-stat.Radius, stat.Y-stat.Radius, stat.X+stat.Radius, stat.Y+stat.Radius)
	dst := image.NewRGBA(image.Rect(0, 0, diameter, diameter))
	drawCrop(dst, numb, crp)

	return dst
}

func drawRadialBar(radius, barWidth, minValue, maxValue, angleStart, angleEnd, angleSep, angleSteps int,
	clockwise bool, value int, text string, withText bool,
	fontName string, fontSize int, fontColor color.Color, barColor color.Color) image.Image {

	if value < minValue {
		value = minValue
	} else if value > maxValue {
		value = maxValue
	}

	pct := float64(value-minValue) / float64(maxValue-minValue)
	rad := func(deg float64) float64 { return deg * math.Pi / 180.0 }

	diameter := 2 * radius
	barImage := image.NewNRGBA(image.Rect(0, 0, diameter, diameter))

	rCenter := float64(radius)
	arcRadius := rCenter - float64(barWidth)/2.0
	angleStartMod := float64(angleStart % 361)
	angleEndMod := float64(angleEnd % 361)

	if clockwise {
		ecart := 0.0
		if angleEndMod < angleStartMod {
			ecart = 360 - angleStartMod + angleEndMod
		} else {
			ecart = angleEndMod - angleStartMod
		}
		if angleSep == 0 {
			aE := angleStartMod + pct*ecart
			drawArc(barImage, rCenter, rCenter, arcRadius, rad(angleStartMod), rad(aE), barWidth, barColor)
		} else {
			aE := angleStartMod + pct*ecart
			angleComplet := ecart / float64(angleSteps)
			etapes := int((aE - angleStartMod) / angleComplet)
			for i := 0; i < etapes; i++ {
				s := angleStartMod + float64(i)*angleComplet
				e := angleStartMod + float64(i+1)*angleComplet - float64(angleSep)
				drawArc(barImage, rCenter, rCenter, arcRadius, rad(s), rad(e), barWidth, barColor)
			}
			s := angleStartMod + float64(etapes)*angleComplet
			drawArc(barImage, rCenter, rCenter, arcRadius, rad(s), rad(aE), barWidth, barColor)
		}
	} else {
		ecart := 0.0
		if angleEndMod < angleStartMod {
			ecart = angleStartMod - angleEndMod
		} else {
			ecart = 360 - angleEndMod + angleStartMod
		}
		if angleSep == 0 {
			aS := angleStartMod - pct*ecart
			drawArc(barImage, rCenter, rCenter, arcRadius, rad(aS), rad(angleStartMod), barWidth, barColor)
		} else {
			aS := angleStartMod - pct*ecart
			angleComplet := ecart / float64(angleSteps)
			etapes := int((angleStartMod - aS) / angleComplet)
			for i := 0; i < etapes; i++ {
				e := angleStartMod - float64(i)*angleComplet
				s := angleStartMod - float64(i+1)*angleComplet + float64(angleSep)
				drawArc(barImage, rCenter, rCenter, arcRadius, rad(s), rad(e), barWidth, barColor)
			}
			e := angleStartMod - float64(etapes)*angleComplet
			drawArc(barImage, rCenter, rCenter, arcRadius, rad(aS), rad(e), barWidth, barColor)
		}
	}

	if withText {
		if text == "" {
			text = fmt.Sprintf("%d%%", int(pct*100+0.5))
		}
		fontBytes, _ := os.ReadFile("./res/fonts/" + fontName)
		f, _ := opentype.Parse(fontBytes)
		face, _ := opentype.NewFace(f, &opentype.FaceOptions{Size: float64(fontSize), DPI: 72})

		w := measureString(face, text)
		ascent := fixedAscent(face)
		descent := fixedDescent(face)
		totalH := ascent + descent

		drawX := fixed.Int26_6(int((float64(radius) - w/2) * 64))
		drawY := fixed.I(radius) + totalH/2 - descent

		d := &font.Drawer{
			Dst:  barImage,
			Src:  image.NewUniform(fontColor),
			Face: face,
			Dot:  fixed.Point26_6{X: drawX, Y: drawY},
		}
		d.DrawString(text)
	}

	return barImage
}

func (b *Builder) saveImage(img image.Image, file string) {
	f, err := os.Create(file)
	if err != nil {
		b.log.Infof("error creating file: %s\n", err)
		return
	}
	defer f.Close()
	err = png.Encode(f, img)
	if err != nil {
		b.log.Infof("error saving file: %s\n", err)
	}
}

// =====================================================================
// Helper functions (pure Go, no CGO)
// =====================================================================

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

// measureString returns the width of the string in float64 pixels.
func measureString(face font.Face, s string) float64 {
	advance := font.MeasureString(face, s)
	return fixedToFloat(advance)
}

// measureHeight returns the line height (ascent + descent) in float64.
func measureHeight(face font.Face) float64 {
	m := face.Metrics()
	return fixedToFloat(m.Ascent + m.Descent)
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
