package ui

import (
	"image"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/alexwbaule/turing-screen/internal/theme"
)

func getPlaceholderForField(path string) string {
	upperPath := strings.ToUpper(path)
	if strings.Contains(upperPath, "PERCENT") {
		return "100%"
	}
	if strings.Contains(upperPath, "HOUR") {
		return "23:59:59"
	}
	if strings.Contains(upperPath, "DAY") {
		return "28/12/2025"
	}
	if strings.Contains(upperPath, "TEMP") {
		return "99°C"
	}
	if strings.Contains(upperPath, "FREQ") {
		return "5.8GHz"
	}
	if strings.Contains(upperPath, "UPLOAD") || strings.Contains(upperPath, "DOWNLOAD") {
		return "999.9MB/s"
	}
	return "8888"
}

type DraggableWidget struct {
	widget.BaseWidget
	YAMLPath    string
	TextData    *theme.Text
	onTapped    func(dw *DraggableWidget)
	onDragEnded func(dw *DraggableWidget)
	fontCache   *FontCache

	raster    *canvas.Raster
	selection *canvas.Rectangle
	hitbox    *canvas.Rectangle
}

func NewDraggableWidget(data *theme.Text, path string, fc *FontCache, tapped func(dw *DraggableWidget), dragEnd func(dw *DraggableWidget)) *DraggableWidget {
	dw := &DraggableWidget{
		YAMLPath:    path,
		TextData:    data,
		onTapped:    tapped,
		onDragEnded: dragEnd,
		fontCache:   fc,
	}

	dw.raster = canvas.NewRaster(dw.renderImage)
	dw.selection = canvas.NewRectangle(color.Transparent)
	dw.selection.StrokeColor = color.Gray{Y: 150}
	dw.selection.StrokeWidth = 1
	dw.selection.Hide()

	dw.hitbox = canvas.NewRectangle(color.Transparent)

	dw.ExtendBaseWidget(dw)
	return dw
}

func (dw *DraggableWidget) renderImage(w, h int) image.Image {
	// Use a copy of data for rendering to avoid mutating the original
	renderData := *dw.TextData
	if renderData.Text == "" {
		// Use Placeholder from theme if available, otherwise infer from YAML path
		if renderData.Placeholder != "" {
			renderData.Text = renderData.Placeholder
		} else {
			renderData.Text = getPlaceholderForField(dw.YAMLPath)
		}
	}
	return renderGGText(&renderData, dw.fontCache)
}

func (dw *DraggableWidget) Tapped(_ *fyne.PointEvent) {
	if dw.onTapped != nil {
		dw.onTapped(dw)
	}
}

func (dw *DraggableWidget) Dragged(e *fyne.DragEvent) {
	potentialPos := dw.Position().Add(e.Dragged)
	widgetSize := dw.Size()
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
	dw.Move(fyne.NewPos(finalX, finalY))
}
func (dw *DraggableWidget) DragEnd() {
	finalPos := dw.Position()
	widgetWidth := dw.Size().Width

	// Convert visual position back to YAML X based on alignment
	align := strings.ToUpper(dw.TextData.Align)
	switch align {
	case "CENTER":
		dw.TextData.X = int(finalPos.X + widgetWidth/2)
	case "RIGHT":
		dw.TextData.X = int(finalPos.X + widgetWidth)
	default: // LEFT
		dw.TextData.X = int(finalPos.X)
	}
	dw.TextData.Y = int(finalPos.Y)

	if dw.onDragEnded != nil {
		dw.onDragEnded(dw)
	}
}
func (dw *DraggableWidget) Select() {
	dw.selection.Show()
	dw.selection.Refresh()
}
func (dw *DraggableWidget) Deselect() {
	dw.selection.Hide()
	dw.selection.Refresh()
}
func (dw *DraggableWidget) CreateRenderer() fyne.WidgetRenderer {
	r := &draggableWidgetRenderer{dw: dw}
	r.Refresh()
	return r
}

type draggableWidgetRenderer struct {
	dw *DraggableWidget
}

func (r *draggableWidgetRenderer) Destroy() {}
func (r *draggableWidgetRenderer) Layout(size fyne.Size) {
	r.dw.raster.Resize(size)
	r.dw.selection.Resize(size)
	r.dw.hitbox.Resize(size)
}
func (r *draggableWidgetRenderer) MinSize() fyne.Size {
	img := r.dw.renderImage(0, 0)
	if img != nil {
		bounds := img.Bounds()
		w := float32(bounds.Dx())
		h := float32(bounds.Dy())
		if w > 0 && h > 0 {
			return fyne.NewSize(w, h)
		}
	}
	return fyne.NewSize(80, 24) // Fallback visible size
}
func (r *draggableWidgetRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.dw.hitbox, r.dw.raster, r.dw.selection}
}
func (r *draggableWidgetRenderer) Refresh() {
	canvas.Refresh(r.dw.raster)
	r.dw.selection.Refresh()
}
