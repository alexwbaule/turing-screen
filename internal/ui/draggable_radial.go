package ui

import (
	"image"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/alexwbaule/turing-screen/internal/theme"
)

type DraggableRadial struct {
	widget.BaseWidget
	YAMLPath    string
	RadialData  *theme.Radial
	onTapped    func(dr *DraggableRadial)
	onDragEnded func(dr *DraggableRadial)
	fontCache   *FontCache
	raster      *canvas.Raster
	selection   *canvas.Rectangle
	hitbox      *canvas.Rectangle
}

func NewDraggableRadial(data *theme.Radial, path string, fc *FontCache, tapped func(dr *DraggableRadial), dragEnd func(dr *DraggableRadial)) *DraggableRadial {
	dr := &DraggableRadial{
		YAMLPath:    path,
		RadialData:  data,
		onTapped:    tapped,
		onDragEnded: dragEnd,
		fontCache:   fc,
	}
	dr.raster = canvas.NewRaster(dr.renderImage)
	dr.selection = canvas.NewRectangle(color.Transparent)
	dr.selection.StrokeColor = color.Gray{Y: 150}
	dr.selection.StrokeWidth = 1
	dr.selection.Hide()
	dr.hitbox = canvas.NewRectangle(color.Transparent)
	dr.ExtendBaseWidget(dr)
	return dr
}

func (dr *DraggableRadial) renderImage(w, h int) image.Image {
	return renderGGRadial(dr.RadialData, dr.fontCache)
}

func (dr *DraggableRadial) Tapped(_ *fyne.PointEvent) {
	if dr.onTapped != nil {
		dr.onTapped(dr)
	}
}

func (dr *DraggableRadial) Dragged(e *fyne.DragEvent) {
	potentialPos := dr.Position().Add(e.Dragged)
	widgetSize := dr.Size()
	canvasWidth := currentCanvasWidth
	canvasHeight := currentCanvasHeight
	finalX := potentialPos.X
	if finalX < 0 {
		finalX = 0
	}
	if finalX+widgetSize.Width > canvasWidth {
		finalX = canvasWidth - widgetSize.Width
	}
	finalY := potentialPos.Y
	if finalY < 0 {
		finalY = 0
	}
	if finalY+widgetSize.Height > canvasHeight {
		finalY = canvasHeight - widgetSize.Height
	}
	dr.Move(fyne.NewPos(finalX, finalY))
}
func (dr *DraggableRadial) DragEnd() {
	pos := dr.Position()
	radius := float32(dr.RadialData.Radius)
	dr.RadialData.X = int(pos.X + radius)
	dr.RadialData.Y = int(pos.Y + radius)
	if dr.onDragEnded != nil {
		dr.onDragEnded(dr)
	}
}
func (dr *DraggableRadial) Select() {
	dr.selection.Show()
	dr.selection.Refresh()
}
func (dr *DraggableRadial) Deselect() {
	dr.selection.Hide()
	dr.selection.Refresh()
}
func (dr *DraggableRadial) CreateRenderer() fyne.WidgetRenderer {
	r := &draggableRadialRenderer{dr: dr}
	r.Refresh()
	return r
}

type draggableRadialRenderer struct {
	dr *DraggableRadial
}

func (r *draggableRadialRenderer) Destroy() {}
func (r *draggableRadialRenderer) Layout(size fyne.Size) {
	r.dr.raster.Resize(size)
	r.dr.selection.Resize(size)
	r.dr.hitbox.Resize(size)
}
func (r *draggableRadialRenderer) MinSize() fyne.Size {
	diameter := float32(r.dr.RadialData.Radius * 2)
	return fyne.NewSize(diameter, diameter)
}
func (r *draggableRadialRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.dr.hitbox, r.dr.raster, r.dr.selection}
}
func (r *draggableRadialRenderer) Refresh() {
	canvas.Refresh(r.dr.raster)
	r.dr.selection.Refresh()
}
