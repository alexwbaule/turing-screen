package ui

import (
	"image"
	"image/color"

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
	raster      *canvas.Raster
	selection   *canvas.Rectangle
	hitbox      *canvas.Rectangle
}

func NewDraggableChart(data *theme.Chart, path string, tapped func(dc *DraggableChart), dragEnd func(dc *DraggableChart)) *DraggableChart {
	dc := &DraggableChart{
		YAMLPath:    path,
		ChartData:   data,
		onTapped:    tapped,
		onDragEnded: dragEnd,
	}
	dc.raster = canvas.NewRaster(dc.renderImage)
	dc.selection = canvas.NewRectangle(color.Transparent)
	dc.selection.StrokeColor = color.Gray{Y: 150}
	dc.selection.StrokeWidth = 1
	dc.selection.Hide()
	dc.hitbox = canvas.NewRectangle(color.Transparent)
	dc.ExtendBaseWidget(dc)
	return dc
}

func (dc *DraggableChart) renderImage(w, h int) image.Image {
	return renderGGChart(dc.ChartData)
}

func (dc *DraggableChart) Tapped(_ *fyne.PointEvent) {
	if dc.onTapped != nil {
		dc.onTapped(dc)
	}
}

func (dc *DraggableChart) Dragged(e *fyne.DragEvent) {
	potentialPos := dc.Position().Add(e.Dragged)
	widgetSize := dc.Size()
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

func (dc *DraggableChart) Select() {
	dc.selection.Show()
	dc.selection.Refresh()
}

func (dc *DraggableChart) Deselect() {
	dc.selection.Hide()
	dc.selection.Refresh()
}

func (dc *DraggableChart) CreateRenderer() fyne.WidgetRenderer {
	r := &draggableChartRenderer{dc: dc}
	r.Refresh()
	return r
}

type draggableChartRenderer struct {
	dc *DraggableChart
}

func (r *draggableChartRenderer) Destroy() {}

func (r *draggableChartRenderer) Layout(size fyne.Size) {
	r.dc.raster.Resize(size)
	r.dc.selection.Resize(size)
	r.dc.hitbox.Resize(size)
}

func (r *draggableChartRenderer) MinSize() fyne.Size {
	return fyne.NewSize(float32(r.dc.ChartData.Width), float32(r.dc.ChartData.Height))
}

func (r *draggableChartRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.dc.hitbox, r.dc.raster, r.dc.selection}
}

func (r *draggableChartRenderer) Refresh() {
	canvas.Refresh(r.dc.raster)
	r.dc.selection.Refresh()
}
