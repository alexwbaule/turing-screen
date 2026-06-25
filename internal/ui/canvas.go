package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// Current canvas dimensions (updated by SetCanvasSize)
var currentCanvasWidth float32 = 1280
var currentCanvasHeight float32 = 720

func buildCanvas(app *EditorApp) fyne.CanvasObject {
	bgColor := color.NRGBA{R: 0x42, G: 0x42, B: 0x42, A: 0xff} // #424242
	app.canvasPlaceholder = canvas.NewRectangle(bgColor)
	app.canvasPlaceholder.SetMinSize(fyne.NewSize(currentCanvasWidth, currentCanvasHeight))

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

	// Outer background sized to the current canvas dimensions (updated by SetCanvasSize).
	outerBg := canvas.NewRectangle(color.NRGBA{R: 0x1e, G: 0x1e, B: 0x1e, A: 0xff})
	outerBg.SetMinSize(fyne.NewSize(currentCanvasWidth, currentCanvasHeight))
	app.outerBg = outerBg

	return container.NewStack(outerBg, container.NewCenter(app.canvasStack))
}

// SetCanvasSize resizes the canvas to match the theme's native resolution.
// w and h are the theme dimensions (e.g. 1280×720 for landscape, 720×1280 for portrait).
// Falls back to orientation-based defaults when w or h is zero (old themes without WIDTH/HEIGHT).
func (e *EditorApp) SetCanvasSize(w, h int, orientation string) {
	if w <= 0 || h <= 0 {
		o := strings.ToLower(orientation)
		if o == "portrait" || o == "reverse_portrait" {
			w, h = 720, 1280
		} else {
			w, h = 1280, 720
		}
	}
	e.canvasWidth = w
	e.canvasHeight = h
	currentCanvasWidth = float32(w)
	currentCanvasHeight = float32(h)
	e.videoPlayer.SetSize(w, h)
	e.canvasPlaceholder.SetMinSize(fyne.NewSize(float32(w), float32(h)))
	e.canvasPlaceholder.Refresh()
	if e.outerBg != nil {
		e.outerBg.SetMinSize(fyne.NewSize(float32(w), float32(h)))
		e.outerBg.Refresh()
	}
	e.canvasStack.Refresh()
	// Resize the editor window to fit the theme canvas plus fixed panel widths.
	if e.editorWindow != nil {
		// left panels (toolbox+layers) ~220px, right panel (properties) ~350px
		e.editorWindow.Resize(fyne.NewSize(float32(w)+570, float32(h)+60))
	}
}

// SetCanvasOrientation is kept for backward compatibility.
func (e *EditorApp) SetCanvasOrientation(orientation string) {
	e.SetCanvasSize(0, 0, orientation)
}
