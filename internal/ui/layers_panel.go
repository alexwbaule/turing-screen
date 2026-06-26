package ui

import (
	"fmt"
	"image"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/alexwbaule/turing-screen/internal/theme"
)

// layerEntry represents one item in the layers list (background layer or canvas widget).
type layerEntry struct {
	label    string
	isLayer  bool   // true = static_image layer, false = canvas widget
	layerKey string // key in StaticImages (if isLayer)
	widget   Selectable
}

// LayersPanel manages background layers and provides a list of all visible components
// for selection (solves overlapping widget problem).
//
// Uses a simple VBox of buttons instead of widget.List. For the small number of
// entries typical in a theme (5-20), a VBox is cheaper: selection changes only
// refresh 2 buttons instead of the entire visible list.
type LayersPanel struct {
	app           *EditorApp
	container     *fyne.Container
	itemsBox      *fyne.Container
	scroll        *container.Scroll
	entries       []layerEntry
	buttons       []*widget.Button
	layerOrder    []string // ordered layer keys (determines z-order for display)
	selectedIndex int
	syncing       bool // prevents recursive selection loop
}

func NewLayersPanel(app *EditorApp) *LayersPanel {
	lp := &LayersPanel{app: app, selectedIndex: -1}

	lp.itemsBox = container.NewVBox()
	lp.scroll = container.NewVScroll(lp.itemsBox)
	lp.scroll.SetMinSize(fyne.NewSize(200, 1))

	addBtn := widget.NewButtonWithIcon("", fyneTheme.ContentAddIcon(), func() {
		lp.addLayer()
	})
	removeBtn := widget.NewButtonWithIcon("", fyneTheme.DeleteIcon(), func() {
		lp.removeSelected()
	})
	upBtn := widget.NewButtonWithIcon("", fyneTheme.MoveUpIcon(), func() {
		lp.moveUp()
	})
	downBtn := widget.NewButtonWithIcon("", fyneTheme.MoveDownIcon(), func() {
		lp.moveDown()
	})

	buttons := container.NewHBox(addBtn, removeBtn, upBtn, downBtn)
	header := widget.NewLabelWithStyle("Layers", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	lp.container = container.NewBorder(
		container.NewVBox(header, buttons),
		nil, nil, nil,
		lp.scroll,
	)

	return lp
}

// Refresh rebuilds the entries list from layerOrder + canvas widgets and recreates
// the button list. Only called on structural changes (add/remove/reorder), not on
// selection changes.
func (lp *LayersPanel) Refresh() {
	lp.entries = nil

	// 0. Video (if present, always first/bottom)
	if lp.app.currentTheme.VideoPlay != nil {
		lp.entries = append(lp.entries, layerEntry{
			label:    fmt.Sprintf("🎬 VIDEO (%s)", filepath.Base(lp.app.currentTheme.VideoPlay.Path)),
			isLayer:  true,
			layerKey: "__VIDEO__",
		})
	}

	// 1. Background layers in display order
	for _, key := range lp.layerOrder {
		if img, ok := lp.app.currentTheme.StaticImages[key]; ok {
			lp.entries = append(lp.entries, layerEntry{
				label:    fmt.Sprintf("🖼 %s (%s)", key, filepath.Base(img.Path)),
				isLayer:  true,
				layerKey: key,
			})
		}
	}

	// 2. All canvas widgets
	for _, obj := range lp.app.canvasElements.Objects {
		switch w := obj.(type) {
		case *DraggableWidget:
			lp.entries = append(lp.entries, layerEntry{
				label:  fmt.Sprintf("📝 %s", w.YAMLPath),
				widget: w,
			})
		case *DraggableGraph:
			lp.entries = append(lp.entries, layerEntry{
				label:  fmt.Sprintf("📊 %s", w.YAMLPath),
				widget: w,
			})
		case *DraggableRadial:
			lp.entries = append(lp.entries, layerEntry{
				label:  fmt.Sprintf("🔘 %s", w.YAMLPath),
				widget: w,
			})
		case *DraggableChart:
			lp.entries = append(lp.entries, layerEntry{
				label:  fmt.Sprintf("📈 %s", w.YAMLPath),
				widget: w,
			})
		}
	}

	lp.rebuildButtons()
}

// rebuildButtons creates one button per entry and updates the VBox.
// Called from Refresh() after entries change.
func (lp *LayersPanel) rebuildButtons() {
	lp.buttons = make([]*widget.Button, len(lp.entries))
	objects := make([]fyne.CanvasObject, len(lp.entries))
	for i, entry := range lp.entries {
		idx := i
		b := widget.NewButton(entry.label, func() {
			lp.onTap(idx)
		})
		b.Alignment = widget.ButtonAlignLeading
		if i == lp.selectedIndex {
			b.Importance = widget.HighImportance
		} else {
			b.Importance = widget.LowImportance
		}
		lp.buttons[i] = b
		objects[i] = b
	}
	lp.itemsBox.Objects = objects
	lp.itemsBox.Refresh()
}

// onTap handles a button click in the layers list.
func (lp *LayersPanel) onTap(idx int) {
	if lp.syncing || idx >= len(lp.entries) {
		return
	}
	lp.setSelectionIndex(idx)
	entry := lp.entries[idx]
	if !entry.isLayer && entry.widget != nil {
		lp.app.selectWidget(entry.widget)
	} else if entry.isLayer {
		lp.app.selectWidget(nil)
		lp.app.showLayerProperties(entry.layerKey)
	}
}

// setSelectionIndex changes which button appears selected. Only refreshes the
// two affected buttons (old and new), not the entire list.
func (lp *LayersPanel) setSelectionIndex(idx int) {
	if idx == lp.selectedIndex {
		return
	}
	if lp.selectedIndex >= 0 && lp.selectedIndex < len(lp.buttons) {
		lp.buttons[lp.selectedIndex].Importance = widget.LowImportance
		lp.buttons[lp.selectedIndex].Refresh()
	}
	lp.selectedIndex = idx
	if idx >= 0 && idx < len(lp.buttons) {
		lp.buttons[idx].Importance = widget.HighImportance
		lp.buttons[idx].Refresh()
	}
}

// SyncFromTheme rebuilds layerOrder from the theme's StaticImages (sorted alphabetically).
func (lp *LayersPanel) SyncFromTheme() {
	lp.layerOrder = nil
	if lp.app.currentTheme.StaticImages != nil {
		for k := range lp.app.currentTheme.StaticImages {
			lp.layerOrder = append(lp.layerOrder, k)
		}
		sort.Strings(lp.layerOrder)
	}
}

// ApplyOrderToTheme renames static_images keys to match layerOrder.
// Keys become BACKGROUND, LAYER_2, LAYER_3... preserving the visual order.
func (lp *LayersPanel) ApplyOrderToTheme() {
	if len(lp.layerOrder) == 0 {
		return
	}
	newImages := make(map[string]theme.StaticImage, len(lp.layerOrder))
	for i, oldKey := range lp.layerOrder {
		img, ok := lp.app.currentTheme.StaticImages[oldKey]
		if !ok {
			continue
		}
		var newKey string
		if i == 0 {
			newKey = "BACKGROUND"
		} else {
			newKey = fmt.Sprintf("LAYER_%d", i+1)
		}
		newImages[newKey] = img
	}
	lp.app.currentTheme.StaticImages = newImages
	// Rebuild layerOrder with new keys
	lp.layerOrder = nil
	for k := range newImages {
		lp.layerOrder = append(lp.layerOrder, k)
	}
	sort.Strings(lp.layerOrder)
}

func (lp *LayersPanel) addLayer() {
	fileOpenDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, lp.app.mainWindow)
				return
			}
			if reader == nil {
				return
			}
			path := reader.URI().Path()
			reader.Close()

			file, err := os.Open(path)
			if err != nil {
				dialog.ShowError(err, lp.app.mainWindow)
				return
			}
			img, _, err := image.Decode(file)
			file.Close()
			if err != nil {
				dialog.ShowError(fmt.Errorf("erro ao decodificar imagem: %w", err), lp.app.mainWindow)
				return
			}

			bounds := img.Bounds()
			w, h := bounds.Dx(), bounds.Dy()

			if lp.app.currentTheme.StaticImages == nil {
				lp.app.currentTheme.StaticImages = make(map[string]theme.StaticImage)
			}
			key := lp.generateKey()

			lp.app.currentTheme.StaticImages[key] = theme.StaticImage{
				Path:   filepath.Base(path),
				X:      0,
				Y:      0,
				Width:  w,
				Height: h,
			}
			lp.layerOrder = append(lp.layerOrder, key)

			log.Printf("[LayersPanel] Added layer %q: %s (%dx%d)", key, path, w, h)

			if lp.app.themeDir == "" {
				lp.app.themeDir = filepath.Dir(path)
			}
			lp.app.LoadBackgroundLayersOrdered(lp.app.themeDir, lp.layerOrder)
			lp.Refresh()
		},
		lp.app.mainWindow,
	)
	fileOpenDialog.SetFilter(storage.NewExtensionFileFilter([]string{".png"}))
	fileOpenDialog.Show()
}

func (lp *LayersPanel) removeSelected() {
	if lp.selectedIndex < 0 || lp.selectedIndex >= len(lp.entries) {
		return
	}
	entry := lp.entries[lp.selectedIndex]

	if entry.isLayer {
		dialog.ShowConfirm("Remover Layer", fmt.Sprintf("Remover layer %q?", entry.layerKey), func(confirmed bool) {
			if !confirmed {
				return
			}
			delete(lp.app.currentTheme.StaticImages, entry.layerKey)
			for i, k := range lp.layerOrder {
				if k == entry.layerKey {
					lp.layerOrder = append(lp.layerOrder[:i], lp.layerOrder[i+1:]...)
					break
				}
			}
			log.Printf("[LayersPanel] Removed layer %q", entry.layerKey)
			lp.app.LoadBackgroundLayersOrdered(lp.app.themeDir, lp.layerOrder)
			lp.selectedIndex = -1
			lp.Refresh()
		}, lp.app.mainWindow)
	} else if entry.widget != nil {
		lp.app.selectWidget(entry.widget)
		lp.app.DeleteSelectedWidget()
	}
}

func (lp *LayersPanel) moveUp() {
	if lp.selectedIndex <= 0 || lp.selectedIndex >= len(lp.entries) {
		return
	}
	entry := lp.entries[lp.selectedIndex]
	prevEntry := lp.entries[lp.selectedIndex-1]

	if entry.isLayer && prevEntry.isLayer {
		for i, k := range lp.layerOrder {
			if k == entry.layerKey && i > 0 {
				lp.layerOrder[i], lp.layerOrder[i-1] = lp.layerOrder[i-1], lp.layerOrder[i]
				lp.app.LoadBackgroundLayersOrdered(lp.app.themeDir, lp.layerOrder)
				break
			}
		}
	} else if !entry.isLayer && !prevEntry.isLayer && entry.widget != nil && prevEntry.widget != nil {
		objects := lp.app.canvasElements.Objects
		idxA, idxB := -1, -1
		for i, obj := range objects {
			if obj == entry.widget.(fyne.CanvasObject) {
				idxA = i
			}
			if obj == prevEntry.widget.(fyne.CanvasObject) {
				idxB = i
			}
		}
		if idxA >= 0 && idxB >= 0 {
			objects[idxA], objects[idxB] = objects[idxB], objects[idxA]
			lp.app.canvasElements.Refresh()
		}
	} else {
		return
	}

	lp.selectedIndex--
	lp.Refresh()
	lp.setSelectionIndex(lp.selectedIndex)
}

func (lp *LayersPanel) moveDown() {
	if lp.selectedIndex < 0 || lp.selectedIndex >= len(lp.entries)-1 {
		return
	}
	entry := lp.entries[lp.selectedIndex]
	nextEntry := lp.entries[lp.selectedIndex+1]

	if entry.isLayer && nextEntry.isLayer {
		for i, k := range lp.layerOrder {
			if k == entry.layerKey && i < len(lp.layerOrder)-1 {
				lp.layerOrder[i], lp.layerOrder[i+1] = lp.layerOrder[i+1], lp.layerOrder[i]
				lp.app.LoadBackgroundLayersOrdered(lp.app.themeDir, lp.layerOrder)
				break
			}
		}
	} else if !entry.isLayer && !nextEntry.isLayer && entry.widget != nil && nextEntry.widget != nil {
		objects := lp.app.canvasElements.Objects
		idxA, idxB := -1, -1
		for i, obj := range objects {
			if obj == entry.widget.(fyne.CanvasObject) {
				idxA = i
			}
			if obj == nextEntry.widget.(fyne.CanvasObject) {
				idxB = i
			}
		}
		if idxA >= 0 && idxB >= 0 {
			objects[idxA], objects[idxB] = objects[idxB], objects[idxA]
			lp.app.canvasElements.Refresh()
		}
	} else {
		return
	}

	lp.selectedIndex++
	lp.Refresh()
	lp.setSelectionIndex(lp.selectedIndex)
}

func (lp *LayersPanel) generateKey() string {
	if len(lp.app.currentTheme.StaticImages) == 0 {
		return "BACKGROUND"
	}
	for i := 2; i < 100; i++ {
		key := fmt.Sprintf("LAYER_%d", i)
		if _, exists := lp.app.currentTheme.StaticImages[key]; !exists {
			return key
		}
	}
	return fmt.Sprintf("LAYER_%d", len(lp.app.currentTheme.StaticImages)+1)
}

// SelectByWidget finds the widget in the entries list and highlights its button.
// Only refreshes the two changed buttons (O(1)), not the full list.
func (lp *LayersPanel) SelectByWidget(w Selectable) {
	for i, entry := range lp.entries {
		if entry.widget == w {
			lp.syncing = true
			defer func() { lp.syncing = false }()
			lp.setSelectionIndex(i)
			return
		}
	}
}

// UnselectAll clears the current button selection without triggering callbacks.
func (lp *LayersPanel) UnselectAll() {
	if lp.selectedIndex >= 0 && lp.selectedIndex < len(lp.buttons) {
		lp.buttons[lp.selectedIndex].Importance = widget.LowImportance
		lp.buttons[lp.selectedIndex].Refresh()
	}
	lp.selectedIndex = -1
}
