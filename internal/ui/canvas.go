package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// Current canvas dimensions (updated by SetCanvasOrientation)
var currentCanvasWidth float32 = 800
var currentCanvasHeight float32 = 480

func buildCanvas(app *EditorApp) fyne.CanvasObject {
	bgColor := color.NRGBA{R: 0x42, G: 0x42, B: 0x42, A: 0xff} // #424242
	app.canvasPlaceholder = canvas.NewRectangle(bgColor)
	app.canvasPlaceholder.SetMinSize(fyne.NewSize(800, 480))

	// Stack layers (bottom to top):
	// 1. Placeholder (cinza)
	// 2. Video player raster (streaming from ffmpeg)
	// 3. Background layers container (multiple static_images in order)
	// 4. Theme components (textos, gráficos, etc.)
	app.canvasStack = container.NewStack(
		app.canvasPlaceholder,
		app.videoPlayer.CanvasObject(),
		app.backgroundLayers,
		app.canvasElements,
	)

	// Fixed-size outer container (800x800) so the window doesn't resize
	// when switching between landscape (800x480) and portrait (480x800)
	outerBg := canvas.NewRectangle(color.NRGBA{R: 0x1e, G: 0x1e, B: 0x1e, A: 0xff})
	outerBg.SetMinSize(fyne.NewSize(800, 800))

	return container.NewStack(outerBg, container.NewCenter(app.canvasStack))
}

// SetCanvasOrientation resizes the canvas based on theme orientation.
func (e *EditorApp) SetCanvasOrientation(orientation string) {
	o := strings.ToLower(orientation)
	var w, h float32
	if o == "portrait" || o == "reverse_portrait" {
		w, h = 480, 800
	} else {
		w, h = 800, 480
	}
	e.canvasWidth = int(w)
	e.canvasHeight = int(h)
	currentCanvasWidth = w
	currentCanvasHeight = h
	e.canvasPlaceholder.SetMinSize(fyne.NewSize(w, h))
	e.canvasPlaceholder.Refresh()
	e.canvasStack.Refresh()
}
