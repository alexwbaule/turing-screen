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
type LayersPanel struct {
	app           *EditorApp
	container     *fyne.Container
	list          *widget.List
	entries       []layerEntry
	layerOrder    []string // ordered layer keys (determines z-order for display)
	selectedIndex int
	syncing       bool // prevents recursive selection loop
}

func NewLayersPanel(app *EditorApp) *LayersPanel {
	lp := &LayersPanel{app: app, selectedIndex: -1}

	lp.list = widget.NewList(
		func() int {
			return len(lp.entries)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Item")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < len(lp.entries) {
				label.SetText(lp.entries[id].label)
			}
		},
	)
	lp.list.OnSelected = func(id widget.ListItemID) {
		lp.selectedIndex = id
		if lp.syncing {
			return
		}
		if id < len(lp.entries) {
			entry := lp.entries[id]
			if !entry.isLayer && entry.widget != nil {
				lp.app.selectWidget(entry.widget)
			} else if entry.isLayer {
				// Show layer properties (allow changing file)
				lp.app.selectWidget(nil)
				lp.app.showLayerProperties(entry.layerKey)
			}
		}
	}

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

	scroll := container.NewVScroll(lp.list)
	scroll.SetMinSize(fyne.NewSize(200, 1))

	lp.container = container.NewBorder(
		container.NewVBox(header, buttons),
		nil, nil, nil,
		scroll,
	)

	return lp
}

// Refresh rebuilds the entries list from layerOrder + canvas widgets.
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

	lp.list.Refresh()
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
			// Remove from layerOrder
			for i, k := range lp.layerOrder {
				if k == entry.layerKey {
					lp.layerOrder = append(lp.layerOrder[:i], lp.layerOrder[i+1:]...)
					break
				}
			}
			log.Printf("[LayersPanel] Removed layer %q", entry.layerKey)
			lp.app.LoadBackgroundLayersOrdered(lp.app.themeDir, lp.layerOrder)
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
		// Swap background layers in layerOrder
		for i, k := range lp.layerOrder {
			if k == entry.layerKey && i > 0 {
				lp.layerOrder[i], lp.layerOrder[i-1] = lp.layerOrder[i-1], lp.layerOrder[i]
				lp.app.LoadBackgroundLayersOrdered(lp.app.themeDir, lp.layerOrder)
				break
			}
		}
	} else if !entry.isLayer && !prevEntry.isLayer && entry.widget != nil && prevEntry.widget != nil {
		// Swap canvas widgets z-order
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
		return // can't swap between layers and widgets
	}

	lp.selectedIndex--
	lp.Refresh()
	lp.list.Select(lp.selectedIndex)
}

func (lp *LayersPanel) moveDown() {
	if lp.selectedIndex < 0 || lp.selectedIndex >= len(lp.entries)-1 {
		return
	}
	entry := lp.entries[lp.selectedIndex]
	nextEntry := lp.entries[lp.selectedIndex+1]

	if entry.isLayer && nextEntry.isLayer {
		// Swap background layers in layerOrder
		for i, k := range lp.layerOrder {
			if k == entry.layerKey && i < len(lp.layerOrder)-1 {
				lp.layerOrder[i], lp.layerOrder[i+1] = lp.layerOrder[i+1], lp.layerOrder[i]
				lp.app.LoadBackgroundLayersOrdered(lp.app.themeDir, lp.layerOrder)
				break
			}
		}
	} else if !entry.isLayer && !nextEntry.isLayer && entry.widget != nil && nextEntry.widget != nil {
		// Swap canvas widgets z-order
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
	lp.list.Select(lp.selectedIndex)
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

// SelectByWidget finds the widget in the entries list and selects it.
func (lp *LayersPanel) SelectByWidget(w Selectable) {
	lp.syncing = true
	defer func() { lp.syncing = false }()

	for i, entry := range lp.entries {
		if entry.widget == w {
			lp.selectedIndex = i
			lp.list.Select(i)
			return
		}
	}
}
