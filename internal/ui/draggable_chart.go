package ui

import (
	"image/color"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/alexwbaule/turing-screen/internal/theme"
)

type DraggableChart struct {
	widget.BaseWidget
	YAMLPath    string
	ChartData   *theme.Chart
	onTapped    func(dc *DraggableChart)
	onDragEnded func(dc *DraggableChart)
	img         *canvas.Image
	selection   *canvas.Rectangle
	hitbox      *canvas.Rectangle
	dirty       bool
	renderGen   atomic.Uint64
}

func NewDraggableChart(data *theme.Chart, path string, tapped func(dc *DraggableChart), dragEnd func(dc *DraggableChart)) *DraggableChart {
	dc := &DraggableChart{
		YAMLPath:    path,
		ChartData:   data,
		onTapped:    tapped,
		onDragEnded: dragEnd,
	}
	dc.img = canvas.NewImageFromImage(renderGGChart(data))
	dc.img.FillMode = canvas.ImageFillOriginal
	dc.selection = canvas.NewRectangle(color.Transparent)
	dc.selection.StrokeColor = color.Gray{Y: 150}
	dc.selection.StrokeWidth = 1
	dc.selection.Hide()
	dc.hitbox = canvas.NewRectangle(color.Transparent)
	dc.ExtendBaseWidget(dc)
	return dc
}

func (dc *DraggableChart) Tapped(_ *fyne.PointEvent) {
	if dc.onTapped != nil {
		dc.onTapped(dc)
	}
}

func (dc *DraggableChart) Dragged(e *fyne.DragEvent) {
	potentialPos := dc.Position().Add(e.Dragged)
	widgetSize := dc.Size()
	finalX := potentialPos.X
	if finalX < 0 {
		finalX = 0
	}
	if finalX+widgetSize.Width > currentCanvasWidth {
		finalX = currentCanvasWidth - widgetSize.Width
	}
	finalY := potentialPos.Y
	if finalY < 0 {
		finalY = 0
	}
	if finalY+widgetSize.Height > currentCanvasHeight {
		finalY = currentCanvasHeight - widgetSize.Height
	}
	dc.Move(fyne.NewPos(finalX, finalY))
}

func (dc *DraggableChart) DragEnd() {
	finalPos := dc.Position()
	dc.ChartData.X = int(finalPos.X)
	dc.ChartData.Y = int(finalPos.Y)
	if dc.onDragEnded != nil {
		dc.onDragEnded(dc)
	}
}

func (dc *DraggableChart) Refresh() {
	dc.dirty = true
	dc.BaseWidget.Refresh()
}
func (dc *DraggableChart) Select() {
	dc.selection.Show()
	dc.selection.Refresh()
}
func (dc *DraggableChart) Deselect() {
	dc.selection.Hide()
	dc.selection.Refresh()
}
func (dc *DraggableChart) CreateRenderer() fyne.WidgetRenderer {
	return &draggableChartRenderer{dc: dc}
}

type draggableChartRenderer struct {
	dc *DraggableChart
}

func (r *draggableChartRenderer) Destroy() {}
func (r *draggableChartRenderer) Layout(size fyne.Size) {
	r.dc.img.Resize(size)
	r.dc.selection.Resize(size)
	r.dc.hitbox.Resize(size)
}
func (r *draggableChartRenderer) MinSize() fyne.Size {
	return fyne.NewSize(float32(r.dc.ChartData.Width), float32(r.dc.ChartData.Height))
}
func (r *draggableChartRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.dc.hitbox, r.dc.img, r.dc.selection}
}
func (r *draggableChartRenderer) Refresh() {
	if !r.dc.dirty && r.dc.img.Image != nil {
		return
	}
	r.dc.dirty = false
	gen := r.dc.renderGen.Add(1)
	data := *r.dc.ChartData
	go func() {
		img := renderGGChart(&data)
		fyne.Do(func() {
			if r.dc.renderGen.Load() != gen {
				return
			}
			r.dc.img.Image = img
			canvas.Refresh(r.dc.img)
		})
	}()
}
