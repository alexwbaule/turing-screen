package utils

import (
	"image"
	"image/draw"
)

// Rect represents a bounding box for overlap calculation.
type Rect struct {
	X, Y, W, H int
}

// ToImageRect converts to image.Rectangle.
func (r Rect) ToImageRect() image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
}

// Intersects returns true if two rects have significant overlap (at least minOverlap pixels in both axes).
func (r Rect) Intersects(other Rect) bool {
	const minOverlap = 3 // minimum overlap in pixels to consider "intersecting"

	overlapX := min(r.X+r.W, other.X+other.W) - max(r.X, other.X)
	overlapY := min(r.Y+r.H, other.Y+other.H) - max(r.Y, other.Y)

	return overlapX >= minOverlap && overlapY >= minOverlap
}

// Union returns the smallest rect that contains both rects.
func (r Rect) Union(other Rect) Rect {
	x1 := min(r.X, other.X)
	y1 := min(r.Y, other.Y)
	x2 := max(r.X+r.W, other.X+other.W)
	y2 := max(r.Y+r.H, other.Y+other.H)
	return Rect{X: x1, Y: y1, W: x2 - x1, H: y2 - y1}
}

// ComposeOver draws imgB over imgA at the given offset within imgA's coordinate space.
// Returns a new image containing the composed result.
func ComposeOver(imgA image.Image, imgB image.Image, offsetX, offsetY int) *image.RGBA {
	bounds := imgA.Bounds()
	dst := image.NewRGBA(bounds)
	// Draw base (imgA)
	draw.Draw(dst, bounds, imgA, bounds.Min, draw.Src)
	// Draw overlay (imgB) at offset
	dstRect := image.Rect(offsetX, offsetY, offsetX+imgB.Bounds().Dx(), offsetY+imgB.Bounds().Dy())
	draw.Draw(dst, dstRect, imgB, imgB.Bounds().Min, draw.Over)
	return dst
}

// ComposeInUnion creates a new image sized to the union of two rects,
// draws imgA at its position and imgB over it at its position.
// Returns the composed image and the top-left position of the union rect.
func ComposeInUnion(imgA image.Image, rectA Rect, imgB image.Image, rectB Rect) (*image.RGBA, int, int) {
	union := rectA.Union(rectB)
	dst := image.NewRGBA(image.Rect(0, 0, union.W, union.H))

	// Draw imgA at its relative position within the union
	aOffset := image.Pt(rectA.X-union.X, rectA.Y-union.Y)
	draw.Draw(dst, image.Rect(aOffset.X, aOffset.Y, aOffset.X+imgA.Bounds().Dx(), aOffset.Y+imgA.Bounds().Dy()),
		imgA, imgA.Bounds().Min, draw.Src)

	// Draw imgB over at its relative position
	bOffset := image.Pt(rectB.X-union.X, rectB.Y-union.Y)
	draw.Draw(dst, image.Rect(bOffset.X, bOffset.Y, bOffset.X+imgB.Bounds().Dx(), bOffset.Y+imgB.Bounds().Dy()),
		imgB, imgB.Bounds().Min, draw.Over)

	return dst, union.X, union.Y
}
