package ui

import (
	"image"
	"image/color"
	"strings"
	"sync/atomic"

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

	img       *canvas.Image
	selection *canvas.Rectangle
	hitbox    *canvas.Rectangle
	cachedW   int
	cachedH   int
	dirty     bool
	renderGen atomic.Uint64
}

func NewDraggableWidget(data *theme.Text, path string, fc *FontCache, tapped func(dw *DraggableWidget), dragEnd func(dw *DraggableWidget)) *DraggableWidget {
	dw := &DraggableWidget{
		YAMLPath:    path,
		TextData:    data,
		onTapped:    tapped,
		onDragEnded: dragEnd,
		fontCache:   fc,
		dirty:       true,
	}

	dw.img = canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	dw.img.FillMode = canvas.ImageFillOriginal

	dw.selection = canvas.NewRectangle(color.Transparent)
	dw.selection.StrokeColor = color.Gray{Y: 150}
	dw.selection.StrokeWidth = 1
	dw.selection.Hide()
	dw.hitbox = canvas.NewRectangle(color.Transparent)

	dw.ExtendBaseWidget(dw)
	return dw
}

func (dw *DraggableWidget) buildImage() image.Image {
	renderData := *dw.TextData
	if renderData.Text == "" {
		if renderData.Placeholder != "" {
			renderData.Text = renderData.Placeholder
		} else {
			renderData.Text = getPlaceholderForField(dw.YAMLPath)
		}
	}
	img := renderGGText(&renderData, dw.fontCache)
	b := img.Bounds()
	dw.cachedW = b.Dx()
	dw.cachedH = b.Dy()
	return img
}

func (dw *DraggableWidget) Tapped(_ *fyne.PointEvent) {
	if dw.onTapped != nil {
		dw.onTapped(dw)
	}
}

func (dw *DraggableWidget) Dragged(e *fyne.DragEvent) {
	potentialPos := dw.Position().Add(e.Dragged)
	widgetSize := dw.Size()
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
	dw.Move(fyne.NewPos(finalX, finalY))
}

func (dw *DraggableWidget) DragEnd() {
	finalPos := dw.Position()
	widgetWidth := dw.Size().Width
	align := strings.ToUpper(dw.TextData.Align)
	switch align {
	case "CENTER":
		dw.TextData.X = int(finalPos.X + widgetWidth/2)
	case "RIGHT":
		dw.TextData.X = int(finalPos.X + widgetWidth)
	default:
		dw.TextData.X = int(finalPos.X)
	}
	dw.TextData.Y = int(finalPos.Y)
	if dw.onDragEnded != nil {
		dw.onDragEnded(dw)
	}
}

func (dw *DraggableWidget) Refresh() {
	dw.dirty = true
	dw.BaseWidget.Refresh()
}
func (dw *DraggableWidget) Select() {
	if dw.selection.Visible() {
		return
	}
	dw.selection.Show()
	dw.selection.Refresh()
}
func (dw *DraggableWidget) Deselect() {
	if !dw.selection.Visible() {
		return
	}
	dw.selection.Hide()
	dw.selection.Refresh()
}
func (dw *DraggableWidget) CreateRenderer() fyne.WidgetRenderer {
	r := &draggableWidgetRenderer{dw: dw}
	return r
}

type draggableWidgetRenderer struct {
	dw *DraggableWidget
}

func (r *draggableWidgetRenderer) Destroy() {}
func (r *draggableWidgetRenderer) Layout(size fyne.Size) {
	r.dw.img.Resize(size)
	r.dw.selection.Resize(size)
	r.dw.hitbox.Resize(size)
}
func (r *draggableWidgetRenderer) MinSize() fyne.Size {
	if r.dw.cachedW > 0 && r.dw.cachedH > 0 {
		return fyne.NewSize(float32(r.dw.cachedW), float32(r.dw.cachedH))
	}
	return fyne.NewSize(80, 24)
}
func (r *draggableWidgetRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.dw.hitbox, r.dw.img, r.dw.selection}
}
func (r *draggableWidgetRenderer) Refresh() {
	if !r.dw.dirty && r.dw.img.Image != nil {
		return
	}
	r.dw.dirty = false
	gen := r.dw.renderGen.Add(1)
	textData := *r.dw.TextData
	if textData.Text == "" {
		if textData.Placeholder != "" {
			textData.Text = textData.Placeholder
		} else {
			textData.Text = getPlaceholderForField(r.dw.YAMLPath)
		}
	}
	fc := r.dw.fontCache
	go func() {
		img := renderGGText(&textData, fc)
		b := img.Bounds()
		fyne.Do(func() {
			if r.dw.renderGen.Load() != gen {
				return
			}
			r.dw.img.Image = img
			r.dw.cachedW = b.Dx()
			r.dw.cachedH = b.Dy()
			canvas.Refresh(r.dw.img)
			r.dw.Resize(r.dw.MinSize())
		})
	}()
}
