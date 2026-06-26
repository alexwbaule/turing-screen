package ui

import (
	"image/color"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/alexwbaule/turing-screen/internal/theme"
)

type DraggableGraph struct {
	widget.BaseWidget
	YAMLPath    string
	GraphData   *theme.Graph
	onTapped    func(dg *DraggableGraph)
	onDragEnded func(dg *DraggableGraph)
	img         *canvas.Image
	selection   *canvas.Rectangle
	hitbox      *canvas.Rectangle
	dirty       bool
	renderGen   atomic.Uint64
}

func NewDraggableGraph(data *theme.Graph, path string, tapped func(dg *DraggableGraph), dragEnd func(dg *DraggableGraph)) *DraggableGraph {
	dg := &DraggableGraph{
		YAMLPath:    path,
		GraphData:   data,
		onTapped:    tapped,
		onDragEnded: dragEnd,
		dirty:       true,
	}
	dg.img = canvas.NewImageFromImage(nil)
	dg.img.FillMode = canvas.ImageFillOriginal
	dg.selection = canvas.NewRectangle(color.Transparent)
	dg.selection.StrokeColor = color.Gray{Y: 150}
	dg.selection.StrokeWidth = 1
	dg.selection.Hide()
	dg.hitbox = canvas.NewRectangle(color.Transparent)
	dg.ExtendBaseWidget(dg)
	return dg
}

func (dg *DraggableGraph) Tapped(_ *fyne.PointEvent) {
	if dg.onTapped != nil {
		dg.onTapped(dg)
	}
}

func (dg *DraggableGraph) Dragged(e *fyne.DragEvent) {
	potentialPos := dg.Position().Add(e.Dragged)
	widgetSize := dg.Size()
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
	dg.Move(fyne.NewPos(finalX, finalY))
}

func (dg *DraggableGraph) DragEnd() {
	finalPos := dg.Position()
	dg.GraphData.X = int(finalPos.X)
	dg.GraphData.Y = int(finalPos.Y)
	if dg.onDragEnded != nil {
		dg.onDragEnded(dg)
	}
}

func (dg *DraggableGraph) Refresh() {
	dg.dirty = true
	dg.BaseWidget.Refresh()
}
func (dg *DraggableGraph) Select() {
	if dg.selection.Visible() {
		return
	}
	dg.selection.Show()
	dg.selection.Refresh()
}
func (dg *DraggableGraph) Deselect() {
	if !dg.selection.Visible() {
		return
	}
	dg.selection.Hide()
	dg.selection.Refresh()
}
func (dg *DraggableGraph) CreateRenderer() fyne.WidgetRenderer {
	return &draggableGraphRenderer{dg: dg}
}

type draggableGraphRenderer struct {
	dg *DraggableGraph
}

func (r *draggableGraphRenderer) Destroy() {}
func (r *draggableGraphRenderer) Layout(size fyne.Size) {
	r.dg.img.Resize(size)
	r.dg.selection.Resize(size)
	r.dg.hitbox.Resize(size)
}
func (r *draggableGraphRenderer) MinSize() fyne.Size {
	return fyne.NewSize(float32(r.dg.GraphData.Width), float32(r.dg.GraphData.Height))
}
func (r *draggableGraphRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.dg.hitbox, r.dg.img, r.dg.selection}
}
func (r *draggableGraphRenderer) Refresh() {
	if !r.dg.dirty && r.dg.img.Image != nil {
		return
	}
	r.dg.dirty = false
	gen := r.dg.renderGen.Add(1)
	data := *r.dg.GraphData
	go func() {
		img := renderGGGraph(&data)
		fyne.Do(func() {
			if r.dg.renderGen.Load() != gen {
				return
			}
			r.dg.img.Image = img
			canvas.Refresh(r.dg.img)
		})
	}()
}
