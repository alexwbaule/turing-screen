package ui

import (
	"fmt"
	"image"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/alexwbaule/turing-screen/internal/theme"
	"github.com/alexwbaule/turing-screen/internal/utils"
	"github.com/spf13/viper"
)

type Selectable interface {
	Select()
	Deselect()
	fyne.Widget
}

type EditorApp struct {
	fyneApp            fyne.App
	mainWindow         fyne.Window
	editorWindow       fyne.Window
	currentTheme       *theme.Theme
	backgroundLayers   *fyne.Container
	videoPlayer        *VideoPlayer
	canvasElements     *fyne.Container
	canvasPlaceholder  *canvas.Rectangle
	canvasStack        *fyne.Container
	propertiesPanel    *PropertiesPanel
	layersPanel        *LayersPanel
	homeScreen         *HomeScreen
	editorContent      fyne.CanvasObject
	selectedWidget     Selectable
	editableStaticText []*theme.Text
	fontCache          *FontCache
	videoPath          string
	backgroundPath     string
	themeDir           string // directory of the currently loaded theme
	themePath          string // full path of the currently loaded theme file (for Save)
	canvasWidth        int    // current canvas width based on orientation
	canvasHeight       int    // current canvas height based on orientation
	statusLabel        *widget.Label
	wsClient           *WSClient
}

// activeWindow returns the editor window if open, otherwise the main window.
func (e *EditorApp) activeWindow() fyne.Window {
	if e.editorWindow != nil {
		return e.editorWindow
	}
	return e.mainWindow
}

func NewEditorApp() *EditorApp {
	a := app.NewWithID("com.alexwbaule.turingthemeditor")
	a.Settings().SetTheme(NewCustomTheme())
	w := a.NewWindow("Turing Theme Editor")

	editor := &EditorApp{
		fyneApp:            a,
		mainWindow:         w,
		backgroundLayers:   container.NewStack(),
		videoPlayer:        NewVideoPlayer(),
		canvasElements:     container.NewWithoutLayout(),
		currentTheme:       &theme.Theme{},
		editableStaticText: []*theme.Text{},
		fontCache:          NewFontCache("res/fonts"),
		canvasWidth:        800,
		canvasHeight:       480,
	}
	editor.propertiesPanel = buildProperties(editor)
	editor.layersPanel = NewLayersPanel(editor)
	return editor
}

func (e *EditorApp) AddComponent(componentType theme.ComponentType) {
	const defaultX, defaultY = 20, 20
	textDefault := theme.Text{X: defaultX, Y: defaultY, FontSize: 22, Font: "jetbrains-mono/JetBrainsMono-Bold.ttf", FontColor: "255,255,255", Show: true, ShowUnit: true}
	graphDefault := theme.Graph{X: defaultX, Y: defaultY, Width: 150, Height: 15, BarColor: "0,255,0", BackgroundColor: "50,50,50", Show: true}
	radialDefault := theme.Radial{X: defaultX, Y: defaultY, Radius: 40, Width: 10, BarColor: "0,255,255", BackgroundColor: "50,50,50", Show: true, AngleStart: 0, AngleEnd: 360, AngleSteps: 36, AngleSep: 2, Clockwise: true}
	chartDefault := theme.Chart{X: defaultX, Y: defaultY, Width: 150, Height: 60, FillColor: "0,200,100", LineColor: "255,255,255", BorderWidth: 1, ColumnWidth: 5, ColumnGap: 1, MinValue: 0, MaxValue: 100, Show: true}

	if e.currentTheme.Stats == nil {
		e.currentTheme.Stats = &theme.Stats{}
	}
	if e.currentTheme.Stats.CPU == nil {
		e.currentTheme.Stats.CPU = &theme.CPU{}
	}
	if e.currentTheme.Stats.GPU == nil {
		e.currentTheme.Stats.GPU = &theme.GPU{}
	}
	if e.currentTheme.Stats.Memory == nil {
		e.currentTheme.Stats.Memory = &theme.Memory{}
	}
	if e.currentTheme.Stats.Memory.Virtual == nil {
		e.currentTheme.Stats.Memory.Virtual = &theme.MemMeasurement{}
	}
	if e.currentTheme.Stats.Memory.Swap == nil {
		e.currentTheme.Stats.Memory.Swap = &theme.MemMeasurement{}
	}
	if e.currentTheme.Stats.Disk == nil {
		e.currentTheme.Stats.Disk = &theme.Disk{}
	}
	if e.currentTheme.Stats.Net == nil {
		e.currentTheme.Stats.Net = &theme.Network{}
	}
	if e.currentTheme.Stats.Net.Wired == nil {
		e.currentTheme.Stats.Net.Wired = &theme.NetworkMeasurement{}
	}
	if e.currentTheme.Stats.Net.Wifi == nil {
		e.currentTheme.Stats.Net.Wifi = &theme.NetworkMeasurement{}
	}
	if e.currentTheme.Stats.Date == nil {
		e.currentTheme.Stats.Date = &theme.DateTime{}
	}

	switch componentType {
	case theme.STATIC_TEXT:
		e.editableStaticText = append(e.editableStaticText, &theme.Text{
			Text: "Texto Estático", X: defaultX, Y: defaultY, FontSize: 14, FontColor: "255,255,255", Show: true,
		})
	// CPU
	case theme.CPU_PERCENTAGE_GRAPH:
		e.currentTheme.Stats.CPU.Percentage = &theme.Measurement{Show: true, Graph: &graphDefault}
	case theme.CPU_PERCENTAGE_TEXT:
		e.currentTheme.Stats.CPU.Percentage = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.CPU_TEMPERATURE_TEXT:
		e.currentTheme.Stats.CPU.Temperature = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.CPU_FREQUENCY_TEXT:
		e.currentTheme.Stats.CPU.Frequency = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.CPU_LOAD_ONE_TEXT:
		if e.currentTheme.Stats.CPU.Load == nil {
			e.currentTheme.Stats.CPU.Load = &theme.Load{}
		}
		e.currentTheme.Stats.CPU.Load.One = &theme.LoadOne{Text: &textDefault}
	case theme.CPU_LOAD_FIVE_TEXT:
		if e.currentTheme.Stats.CPU.Load == nil {
			e.currentTheme.Stats.CPU.Load = &theme.Load{}
		}
		e.currentTheme.Stats.CPU.Load.Five = &theme.LoadFive{Text: &textDefault}
	case theme.CPU_LOAD_FIFTEEN_TEXT:
		if e.currentTheme.Stats.CPU.Load == nil {
			e.currentTheme.Stats.CPU.Load = &theme.Load{}
		}
		e.currentTheme.Stats.CPU.Load.Fifteen = &theme.LoadFifteen{Text: &textDefault}
	case theme.CPU_PERCENTAGE_RADIAL:
		e.currentTheme.Stats.CPU.Percentage = &theme.Measurement{Show: true, Radial: &radialDefault}
	case theme.CPU_PERCENTAGE_CHART:
		e.currentTheme.Stats.CPU.Percentage = &theme.Measurement{Show: true, Chart: &chartDefault}
	// GPU
	case theme.GPU_PERCENTAGE_GRAPH:
		e.currentTheme.Stats.GPU.Percentage = &theme.Measurement{Show: true, Graph: &graphDefault}
	case theme.GPU_TEMPERATURE_TEXT:
		e.currentTheme.Stats.GPU.Temperature = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.GPU_MEMORY_TEXT:
		e.currentTheme.Stats.GPU.Memory = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.GPU_POWER_TEXT:
		e.currentTheme.Stats.GPU.Power = &theme.Measurement{Show: true, Text: &textDefault}
	// Memory Virtual
	case theme.MEMORY_VIRTUAL_PERCENT_TEXT:
		e.currentTheme.Stats.Memory.Virtual.PercentText = &textDefault
		e.currentTheme.Stats.Memory.Virtual.Show = true
	case theme.MEMORY_VIRTUAL_GRAPH:
		e.currentTheme.Stats.Memory.Virtual.Graph = &graphDefault
		e.currentTheme.Stats.Memory.Virtual.Show = true
	case theme.MEMORY_VIRTUAL_RADIAL:
		e.currentTheme.Stats.Memory.Virtual.Radial = &radialDefault
		e.currentTheme.Stats.Memory.Virtual.Show = true
	case theme.MEMORY_VIRTUAL_CHART:
		e.currentTheme.Stats.Memory.Virtual.Chart = &chartDefault
		e.currentTheme.Stats.Memory.Virtual.Show = true
	case theme.MEMORY_VIRTUAL_USED_TEXT:
		e.currentTheme.Stats.Memory.Virtual.Used = &textDefault
		e.currentTheme.Stats.Memory.Virtual.Show = true
	case theme.MEMORY_VIRTUAL_FREE_TEXT:
		e.currentTheme.Stats.Memory.Virtual.Free = &textDefault
		e.currentTheme.Stats.Memory.Virtual.Show = true
	// Memory Swap
	case theme.MEMORY_SWAP_PERCENT_TEXT:
		e.currentTheme.Stats.Memory.Swap.PercentText = &textDefault
		e.currentTheme.Stats.Memory.Swap.Show = true
	case theme.MEMORY_SWAP_GRAPH:
		e.currentTheme.Stats.Memory.Swap.Graph = &graphDefault
		e.currentTheme.Stats.Memory.Swap.Show = true
	case theme.MEMORY_SWAP_RADIAL:
		e.currentTheme.Stats.Memory.Swap.Radial = &radialDefault
		e.currentTheme.Stats.Memory.Swap.Show = true
	case theme.MEMORY_SWAP_CHART:
		e.currentTheme.Stats.Memory.Swap.Chart = &chartDefault
		e.currentTheme.Stats.Memory.Swap.Show = true
	// Disk
	case theme.DISK_USED_PERCENT_TEXT:
		if e.currentTheme.Stats.Disk.Used == nil {
			e.currentTheme.Stats.Disk.Used = &theme.Measurement{}
		}
		e.currentTheme.Stats.Disk.Used.PercentText = &textDefault
		e.currentTheme.Stats.Disk.Used.Show = true
	case theme.DISK_FREE_TEXT:
		if e.currentTheme.Stats.Disk.Free == nil {
			e.currentTheme.Stats.Disk.Free = &theme.Measurement{}
		}
		e.currentTheme.Stats.Disk.Free.Text = &textDefault
		e.currentTheme.Stats.Disk.Free.Show = true
	case theme.DISK_TOTAL_TEXT:
		if e.currentTheme.Stats.Disk.Total == nil {
			e.currentTheme.Stats.Disk.Total = &theme.Measurement{}
		}
		e.currentTheme.Stats.Disk.Total.Text = &textDefault
		e.currentTheme.Stats.Disk.Total.Show = true
	case theme.DISK_TEMPERATURE_TEXT:
		e.currentTheme.Stats.Disk.Temperature = &theme.Measurement{Show: true, Text: &textDefault}
	// Network Ethernet
	case theme.NET_ETH_UPLOAD_TEXT:
		e.currentTheme.Stats.Net.Wired.Upload = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.NET_ETH_DOWNLOAD_TEXT:
		e.currentTheme.Stats.Net.Wired.Download = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.NET_ETH_UPLOADED_TEXT:
		e.currentTheme.Stats.Net.Wired.Uploaded = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.NET_ETH_DOWNLOADED_TEXT:
		e.currentTheme.Stats.Net.Wired.Downloaded = &theme.Measurement{Show: true, Text: &textDefault}
	// Network WiFi
	case theme.NET_WLAN_UPLOAD_TEXT:
		e.currentTheme.Stats.Net.Wifi.Upload = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.NET_WLAN_DOWNLOAD_TEXT:
		e.currentTheme.Stats.Net.Wifi.Download = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.NET_WLAN_UPLOADED_TEXT:
		e.currentTheme.Stats.Net.Wifi.Uploaded = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.NET_WLAN_DOWNLOADED_TEXT:
		e.currentTheme.Stats.Net.Wifi.Downloaded = &theme.Measurement{Show: true, Text: &textDefault}
	// Date/Time
	case theme.TIME_HOUR_TEXT:
		e.currentTheme.Stats.Date.Hour = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.TIME_DAY_TEXT:
		e.currentTheme.Stats.Date.Day = &theme.Measurement{Show: true, Text: &textDefault}
	// CPU extras
	case theme.CPU_FAN_TEXT:
		e.currentTheme.Stats.CPU.Fan = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.CPU_POWER_TEXT:
		e.currentTheme.Stats.CPU.Power = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.CPU_VOLTAGE_TEXT:
		e.currentTheme.Stats.CPU.Voltage = &theme.Measurement{Show: true, Text: &textDefault}
	// GPU extras
	case theme.GPU_FREQUENCY_TEXT:
		e.currentTheme.Stats.GPU.Frequency = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.GPU_VOLTAGE_TEXT:
		e.currentTheme.Stats.GPU.Voltage = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.GPU_FAN_TEXT:
		e.currentTheme.Stats.GPU.Fan = &theme.Measurement{Show: true, Text: &textDefault}
	// Weather
	case theme.WEATHER_TEMPERATURE_TEXT:
		if e.currentTheme.Stats.Weather == nil {
			e.currentTheme.Stats.Weather = &theme.Weather{}
		}
		e.currentTheme.Stats.Weather.Temperature = &theme.Measurement{Show: true, Text: &textDefault}
	case theme.WEATHER_CONDITION_TEXT:
		if e.currentTheme.Stats.Weather == nil {
			e.currentTheme.Stats.Weather = &theme.Weather{}
		}
		condText := theme.Text{X: defaultX, Y: defaultY, FontSize: 22, Font: "jetbrains-mono/JetBrainsMono-Bold.ttf", FontColor: "255,255,255", Show: true, Placeholder: "Clear, 25°C, Wind 10km/h"}
		e.currentTheme.Stats.Weather.Condition = &condText
	// Volume
	case theme.VOLUME_TEXT:
		if e.currentTheme.Stats.Volume == nil {
			e.currentTheme.Stats.Volume = &theme.Volume{}
		}
		e.currentTheme.Stats.Volume.Text = &textDefault
	}
	e.RefreshCanvas()
	e.layersPanel.Refresh()
	e.selectLastWidget()
}

func (e *EditorApp) LoadTheme(path string) {
	log.Printf("========== LOAD THEME INICIADO ==========")
	log.Printf("[LoadTheme] Caminho: %s", path)

	loadedTheme, err := theme.LoadTheme(path)
	if err != nil {
		log.Printf("[LoadTheme] ERRO ao carregar: %v", err)
		dialog.ShowError(err, e.activeWindow())
		return
	}
	e.currentTheme = loadedTheme

	// Set canvas orientation based on theme display
	if loadedTheme.Display != nil && loadedTheme.Display.Orientation != "" {
		e.SetCanvasOrientation(loadedTheme.Display.Orientation)
	} else {
		e.SetCanvasOrientation("landscape")
	}

	// Log do que foi parseado do YAML
	log.Printf("[LoadTheme] Display: %+v", loadedTheme.Display)
	log.Printf("[LoadTheme] StaticImages (%d entradas):", len(loadedTheme.StaticImages))
	for key, img := range loadedTheme.StaticImages {
		log.Printf("[LoadTheme]   StaticImage[%q] = PATH:%q X:%d Y:%d W:%d H:%d", key, img.Path, img.X, img.Y, img.Width, img.Height)
	}
	log.Printf("[LoadTheme] VideoPlay: %v", loadedTheme.VideoPlay != nil)
	if loadedTheme.VideoPlay != nil {
		log.Printf("[LoadTheme]   Video PATH:%q X:%d Y:%d W:%d H:%d", loadedTheme.VideoPlay.Path, loadedTheme.VideoPlay.X, loadedTheme.VideoPlay.Y, loadedTheme.VideoPlay.Width, loadedTheme.VideoPlay.Height)
	}
	log.Printf("[LoadTheme] StaticText (%d entradas):", len(loadedTheme.StaticText))
	for key, txt := range loadedTheme.StaticText {
		log.Printf("[LoadTheme]   StaticText[%q] = TEXT:%q SHOW:%v X:%d Y:%d Font:%q FontSize:%d", key, txt.Text, txt.Show, txt.X, txt.Y, txt.Font, txt.FontSize)
	}
	log.Printf("[LoadTheme] Stats == nil? %v", loadedTheme.Stats == nil)
	if loadedTheme.Stats != nil {
		log.Printf("[LoadTheme]   Stats.CPU == nil? %v", loadedTheme.Stats.CPU == nil)
		log.Printf("[LoadTheme]   Stats.GPU == nil? %v", loadedTheme.Stats.GPU == nil)
		log.Printf("[LoadTheme]   Stats.Memory == nil? %v", loadedTheme.Stats.Memory == nil)
		log.Printf("[LoadTheme]   Stats.Disk == nil? %v", loadedTheme.Stats.Disk == nil)
		log.Printf("[LoadTheme]   Stats.Net == nil? %v", loadedTheme.Stats.Net == nil)
		log.Printf("[LoadTheme]   Stats.Date == nil? %v", loadedTheme.Stats.Date == nil)
	}

	e.selectWidget(nil)
	e.editableStaticText = []*theme.Text{}
	for key := range e.currentTheme.StaticText {
		val := e.currentTheme.StaticText[key]
		val.Show = true
		log.Printf("[LoadTheme] Adicionando static text [%q] Text:%q X:%d Y:%d", key, val.Text, val.X, val.Y)
		e.editableStaticText = append(e.editableStaticText, &val)
	}

	themeDir := filepath.Dir(path)
	e.themeDir = themeDir
	e.themePath = path

	// Reset both layers
	e.videoPath = ""
	e.videoPlayer.Stop()
	e.backgroundPath = ""
	e.backgroundLayers.RemoveAll()

	// Load video if present
	if e.currentTheme.VideoPlay != nil {
		vid := e.currentTheme.VideoPlay
		videoAbsPath := filepath.Join(themeDir, vid.Path)
		log.Printf("[LoadTheme] Video encontrado: PATH=%q => absPath=%q", vid.Path, videoAbsPath)
		if err := e.loadVideoAsBackground(videoAbsPath); err != nil {
			log.Printf("[LoadTheme] ERRO ao carregar vídeo: %v", err)
		}
	}

	// Always load static_images as layers (on top of video if present)
	e.LoadBackgroundLayers(themeDir)
	e.RefreshCanvas()
	e.layersPanel.SyncFromTheme()
	e.layersPanel.Refresh()
	log.Printf("========== LOAD THEME CONCLUÍDO ==========")
}

func (e *EditorApp) RefreshCanvas() {
	log.Printf("[RefreshCanvas] Iniciando refresh...")
	e.canvasElements.RemoveAll()
	log.Printf("[RefreshCanvas] editableStaticText count: %d", len(e.editableStaticText))
	for i, textDataPtr := range e.editableStaticText {
		log.Printf("[RefreshCanvas]   static[%d] Show=%v Text=%q X=%d Y=%d", i, textDataPtr.Show, textDataPtr.Text, textDataPtr.X, textDataPtr.Y)
		if textDataPtr.Show {
			path := fmt.Sprintf("static_texts.[%d]", i)
			e.renderText(textDataPtr, path)
		}
	}
	if e.currentTheme.Stats != nil {
		log.Printf("[RefreshCanvas] Renderizando Stats via reflection...")
		e.renderStruct(reflect.ValueOf(e.currentTheme.Stats), "STATS")
	} else {
		log.Printf("[RefreshCanvas] Stats == nil, pulando")
	}

	// Apply z-order from INDEX fields
	e.applyIndexOrder()

	log.Printf("[RefreshCanvas] Total de objetos no canvas: %d", len(e.canvasElements.Objects))
	e.canvasElements.Refresh()
}

// applyIndexOrder sorts canvas elements by their INDEX field (lower = behind, higher = on top).
func (e *EditorApp) applyIndexOrder() {
	objects := e.canvasElements.Objects
	if len(objects) <= 1 {
		return
	}

	// Stable sort by index
	sort.SliceStable(objects, func(i, j int) bool {
		return getWidgetIndex(objects[i]) < getWidgetIndex(objects[j])
	})
	e.canvasElements.Objects = objects
}

func getWidgetIndex(obj fyne.CanvasObject) int {
	switch w := obj.(type) {
	case *DraggableWidget:
		return w.TextData.Index
	case *DraggableGraph:
		return w.GraphData.Index
	case *DraggableRadial:
		return w.RadialData.Index
	case *DraggableChart:
		return w.ChartData.Index
	}
	return 0
}

func (e *EditorApp) renderStruct(v reflect.Value, currentPath string) {
	if !v.IsValid() {
		log.Printf("[renderStruct] path=%s => valor inválido, pulando", currentPath)
		return
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			log.Printf("[renderStruct] path=%s => ponteiro nil, pulando", currentPath)
			return
		}
		v = v.Elem()
	}

	// Handle Load
	if v.Type() == reflect.TypeOf(theme.Load{}) {
		load := v.Interface().(theme.Load)
		log.Printf("[renderStruct] path=%s => Load detectado (One=%v, Five=%v, Fifteen=%v)", currentPath, load.One != nil, load.Five != nil, load.Fifteen != nil)
		if load.One != nil && load.One.Text != nil && load.One.Text.Show {
			e.renderText(load.One.Text, currentPath+".ONE.TEXT")
		}
		if load.Five != nil && load.Five.Text != nil && load.Five.Text.Show {
			e.renderText(load.Five.Text, currentPath+".FIVE.TEXT")
		}
		if load.Fifteen != nil && load.Fifteen.Text != nil && load.Fifteen.Text.Show {
			e.renderText(load.Fifteen.Text, currentPath+".FIFTEEN.TEXT")
		}
		return
	}

	// Handle Measurement
	if v.Type() == reflect.TypeOf(theme.Measurement{}) {
		measurement := v.Interface().(theme.Measurement)
		log.Printf("[renderStruct] path=%s => Measurement Show=%v Text=%v PercentText=%v Graph=%v Radial=%v Chart=%v",
			currentPath, measurement.Show,
			measurement.Text != nil, measurement.PercentText != nil,
			measurement.Graph != nil, measurement.Radial != nil, measurement.Chart != nil)
		// Render children if Show is true OR if any child exists (turing-screen YAML doesn't set Show on Measurement)
		if measurement.Text != nil && measurement.Text.Show {
			e.renderText(measurement.Text, currentPath+".TEXT")
		}
		if measurement.PercentText != nil && measurement.PercentText.Show {
			e.renderText(measurement.PercentText, currentPath+".PERCENT_TEXT")
		}
		if measurement.Graph != nil && measurement.Graph.Show {
			e.renderGraph(measurement.Graph, currentPath+".GRAPH")
		}
		if measurement.Radial != nil && measurement.Radial.Show {
			e.renderRadial(measurement.Radial, currentPath+".RADIAL")
		}
		if measurement.Chart != nil && measurement.Chart.Show {
			e.renderChart(measurement.Chart, currentPath+".CHART")
		}
		return
	}

	// Handle MemMeasurement
	if v.Type() == reflect.TypeOf(theme.MemMeasurement{}) {
		memMeasurement := v.Interface().(theme.MemMeasurement)
		log.Printf("[renderStruct] path=%s => MemMeasurement Show=%v PercentText=%v Used=%v Free=%v Graph=%v Radial=%v Chart=%v",
			currentPath, memMeasurement.Show,
			memMeasurement.PercentText != nil, memMeasurement.Used != nil, memMeasurement.Free != nil,
			memMeasurement.Graph != nil, memMeasurement.Radial != nil, memMeasurement.Chart != nil)
		// Render children if they exist and have Show=true (independent of MemMeasurement.Show)
		if memMeasurement.PercentText != nil && memMeasurement.PercentText.Show {
			e.renderText(memMeasurement.PercentText, currentPath+".PERCENT_TEXT")
		}
		if memMeasurement.Used != nil && memMeasurement.Used.Show {
			e.renderText(memMeasurement.Used, currentPath+".USED")
		}
		if memMeasurement.Free != nil && memMeasurement.Free.Show {
			e.renderText(memMeasurement.Free, currentPath+".FREE")
		}
		if memMeasurement.Graph != nil && memMeasurement.Graph.Show {
			e.renderGraph(memMeasurement.Graph, currentPath+".GRAPH")
		}
		if memMeasurement.Radial != nil && memMeasurement.Radial.Show {
			e.renderRadial(memMeasurement.Radial, currentPath+".RADIAL")
		}
		if memMeasurement.Chart != nil && memMeasurement.Chart.Show {
			e.renderChart(memMeasurement.Chart, currentPath+".CHART")
		}
		return
	}

	// Handle Weather
	if v.Type() == reflect.TypeOf(theme.Weather{}) {
		w := v.Interface().(theme.Weather)
		if w.Temperature != nil {
			e.renderStruct(reflect.ValueOf(w.Temperature), currentPath+".TEMPERATURE")
		}
		if w.Condition != nil && w.Condition.Show {
			e.renderText(w.Condition, currentPath+".CONDITION")
		}
		return
	}

	// Handle Volume
	if v.Type() == reflect.TypeOf(theme.Volume{}) {
		vol := v.Interface().(theme.Volume)
		if vol.Text != nil && vol.Text.Show {
			e.renderText(vol.Text, currentPath+".TEXT")
		}
		return
	}

	if v.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)
		yamlTag := strings.Split(fieldType.Tag.Get("yaml"), ",")[0]
		if yamlTag == "" || yamlTag == "-" {
			continue
		}
		newPath := currentPath + "." + yamlTag

		// Log field traversal for top-level structs only (to avoid flooding)
		if strings.Count(newPath, ".") <= 2 {
			isNil := field.Kind() == reflect.Ptr && field.IsNil()
			log.Printf("[renderStruct] percorrendo campo %s (kind=%s, nil=%v)", newPath, field.Kind(), isNil)
		}

		e.renderStruct(field, newPath)
	}
}

func (e *EditorApp) renderText(data *theme.Text, path string) {
	log.Printf("[renderText] path=%s Text=%q X=%d Y=%d Font=%q FontSize=%d FontColor=%q BgColor=%q Align=%q",
		path, data.Text, data.X, data.Y, data.Font, data.FontSize, data.FontColor, data.BackgroundColor, data.Align)
	widget := NewDraggableWidget(data, path, e.fontCache,
		func(dw *DraggableWidget) { e.selectWidget(dw) },
		func(dw *DraggableWidget) { e.propertiesPanel.Update(dw) },
	)
	widget.Resize(widget.MinSize())

	// Calculate visual X position based on alignment (same logic as daemon's DrawText)
	posX := float32(data.X)
	widgetWidth := widget.Size().Width
	align := strings.ToUpper(data.Align)
	switch align {
	case "CENTER":
		posX = float32(data.X) - widgetWidth/2
	case "RIGHT":
		posX = float32(data.X) - widgetWidth
	}
	if posX < 0 {
		posX = 0
	}

	widget.Move(fyne.NewPos(posX, float32(data.Y)))
	e.canvasElements.Add(widget)
}

func (e *EditorApp) renderGraph(data *theme.Graph, path string) {
	log.Printf("[renderGraph] path=%s X=%d Y=%d W=%d H=%d BarColor=%q BgColor=%q Outline=%v",
		path, data.X, data.Y, data.Width, data.Height, data.BarColor, data.BackgroundColor, data.BarOutline)
	widget := NewDraggableGraph(data, path,
		func(dg *DraggableGraph) { e.selectWidget(dg) },
		func(dg *DraggableGraph) { e.propertiesPanel.Update(dg) },
	)
	widget.Move(fyne.NewPos(float32(data.X), float32(data.Y)))
	widget.Resize(widget.MinSize())
	e.canvasElements.Add(widget)
}

func (e *EditorApp) renderRadial(data *theme.Radial, path string) {
	log.Printf("[renderRadial] path=%s X=%d Y=%d Radius=%d Width=%d BarColor=%q BgColor=%q ShowText=%v",
		path, data.X, data.Y, data.Radius, data.Width, data.BarColor, data.BackgroundColor, data.ShowText)
	widget := NewDraggableRadial(data, path, e.fontCache,
		func(dr *DraggableRadial) { e.selectWidget(dr) },
		func(dr *DraggableRadial) { e.propertiesPanel.Update(dr) },
	)
	xPos := float32(data.X - data.Radius)
	yPos := float32(data.Y - data.Radius)
	widget.Move(fyne.NewPos(xPos, yPos))
	widget.Resize(widget.MinSize())
	e.canvasElements.Add(widget)
}

func (e *EditorApp) renderChart(data *theme.Chart, path string) {
	log.Printf("[renderChart] path=%s X=%d Y=%d W=%d H=%d FillColor=%q LineColor=%q Border=%d",
		path, data.X, data.Y, data.Width, data.Height, data.FillColor, data.LineColor, data.BorderWidth)
	widget := NewDraggableChart(data, path,
		func(dc *DraggableChart) { e.selectWidget(dc) },
		func(dc *DraggableChart) { e.propertiesPanel.Update(dc) },
	)
	widget.Move(fyne.NewPos(float32(data.X), float32(data.Y)))
	widget.Resize(widget.MinSize())
	e.canvasElements.Add(widget)
}

func (e *EditorApp) NewTheme() {
	log.Println("Criando novo tema...")
	e.currentTheme = &theme.Theme{}
	e.editableStaticText = []*theme.Text{}
	e.videoPath = ""
	e.themePath = ""
	e.themeDir = ""
	e.videoPlayer.Stop()
	canvas.Refresh(e.videoPlayer.CanvasObject())
	e.LoadBackgroundImage("")
	e.SetCanvasOrientation("landscape")
	e.selectWidget(nil)
	e.RefreshCanvas()
	e.layersPanel.SyncFromTheme()
	e.layersPanel.Refresh()
}

func (e *EditorApp) selectLastWidget() {
	if numObjects := len(e.canvasElements.Objects); numObjects > 0 {
		lastWidget := e.canvasElements.Objects[numObjects-1]
		if selectable, ok := lastWidget.(Selectable); ok {
			e.selectWidget(selectable)
		}
	}
}

// selectWidgetByPath finds and selects the widget matching the given YAML path.
func (e *EditorApp) selectWidgetByPath(path string) {
	for _, obj := range e.canvasElements.Objects {
		switch w := obj.(type) {
		case *DraggableWidget:
			if w.YAMLPath == path {
				e.selectWidget(w)
				return
			}
		case *DraggableGraph:
			if w.YAMLPath == path {
				e.selectWidget(w)
				return
			}
		case *DraggableRadial:
			if w.YAMLPath == path {
				e.selectWidget(w)
				return
			}
		case *DraggableChart:
			if w.YAMLPath == path {
				e.selectWidget(w)
				return
			}
		}
	}
}

// showLayerProperties updates the properties panel to show layer/video info.
func (e *EditorApp) showLayerProperties(layerKey string) {
	if layerKey == "__VIDEO__" && e.currentTheme.VideoPlay != nil {
		e.propertiesPanel.ShowLayerInfo("VIDEO", e.currentTheme.VideoPlay.Path, func() {
			// Change video file
			fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil || reader == nil {
					return
				}
				path := reader.URI().Path()
				reader.Close()
				e.currentTheme.VideoPlay.Path = filepath.Base(path)
				e.videoPath = path
				e.loadVideoAsBackground(path)
				e.layersPanel.Refresh()
			}, e.activeWindow())
			fileDialog.Resize(fyne.NewSize(900, 600))
			fileDialog.Show()
		})
	} else if img, ok := e.currentTheme.StaticImages[layerKey]; ok {
		e.propertiesPanel.ShowLayerInfo(layerKey, img.Path, func() {
			// Change image file
			fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil || reader == nil {
					return
				}
				path := reader.URI().Path()
				reader.Close()
				imgData := e.currentTheme.StaticImages[layerKey]
				imgData.Path = filepath.Base(path)
				e.currentTheme.StaticImages[layerKey] = imgData
				if e.themeDir != "" {
					e.LoadBackgroundLayersOrdered(e.themeDir, e.layersPanel.layerOrder)
				}
				e.layersPanel.Refresh()
			}, e.activeWindow())
			fileDialog.Resize(fyne.NewSize(900, 600))
			fileDialog.Show()
		})
	}
}

func (e *EditorApp) selectWidget(w Selectable) {
	if e.selectedWidget != nil {
		e.selectedWidget.Deselect()
	}
	e.selectedWidget = w
	if w == nil {
		e.propertiesPanel.Update(nil)
		e.layersPanel.list.UnselectAll()
		return
	}
	e.selectedWidget.Select()
	e.propertiesPanel.Update(w)

	// Sync selection in layers panel
	e.layersPanel.SelectByWidget(w)
}

func (e *EditorApp) DeleteSelectedWidget() {
	if e.selectedWidget == nil {
		return
	}
	dialog.ShowConfirm("Confirmar Exclusão", "Você tem certeza que deseja excluir este componente?", func(confirmed bool) {
		if !confirmed {
			return
		}
		switch v := e.selectedWidget.(type) {
		case *DraggableWidget:
			v.TextData.Show = false
		case *DraggableGraph:
			v.GraphData.Show = false
		case *DraggableRadial:
			v.RadialData.Show = false
		case *DraggableChart:
			v.ChartData.Show = false
		}
		e.selectWidget(nil)
		e.RefreshCanvas()
		e.layersPanel.Refresh()
	}, e.activeWindow())
}

func (e *EditorApp) LoadBackgroundImage(path string) {
	log.Printf("[LoadBackgroundImage] path=%q", path)
	if path == "" {
		e.backgroundLayers.RemoveAll()
		e.backgroundPath = ""
		log.Printf("[LoadBackgroundImage] Limpo (sem background)")
	} else {
		file, err := os.Open(path)
		if err != nil {
			log.Printf("[LoadBackgroundImage] ERRO ao abrir arquivo: %v", err)
			dialog.ShowError(fmt.Errorf("erro ao abrir imagem de fundo: %w", err), e.activeWindow())
			return
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			log.Printf("[LoadBackgroundImage] ERRO ao decodificar imagem: %v", err)
			dialog.ShowError(fmt.Errorf("erro ao decodificar imagem de fundo: %w", err), e.activeWindow())
			return
		}

		bgCanvas := canvas.NewImageFromImage(img)
		bgCanvas.FillMode = canvas.ImageFillOriginal
		e.backgroundLayers.RemoveAll()
		e.backgroundLayers.Add(bgCanvas)
		e.backgroundPath = path
		log.Printf("[LoadBackgroundImage] Imagem carregada: %s (bounds=%v)", path, img.Bounds())
	}
	e.backgroundLayers.Refresh()
}

// LoadBackgroundLayers loads multiple static_images as layers in alphabetical order.
func (e *EditorApp) LoadBackgroundLayers(themeDir string) {
	e.backgroundLayers.RemoveAll()

	if e.currentTheme.StaticImages == nil || len(e.currentTheme.StaticImages) == 0 {
		return
	}

	// Sort keys alphabetically (determines z-order)
	keys := make([]string, 0, len(e.currentTheme.StaticImages))
	for k := range e.currentTheme.StaticImages {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		imgData := e.currentTheme.StaticImages[key]
		absPath := filepath.Join(themeDir, imgData.Path)
		log.Printf("[LoadBackgroundLayers] Layer %q: %s at (%d,%d) %dx%d", key, absPath, imgData.X, imgData.Y, imgData.Width, imgData.Height)

		file, err := os.Open(absPath)
		if err != nil {
			log.Printf("[LoadBackgroundLayers] ERRO ao abrir %s: %v", absPath, err)
			continue
		}

		img, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			log.Printf("[LoadBackgroundLayers] ERRO ao decodificar %s: %v", absPath, err)
			continue
		}

		layerCanvas := canvas.NewImageFromImage(img)
		layerCanvas.FillMode = canvas.ImageFillOriginal
		// Position the layer
		layerCanvas.Move(fyne.NewPos(float32(imgData.X), float32(imgData.Y)))
		layerCanvas.Resize(fyne.NewSize(float32(img.Bounds().Dx()), float32(img.Bounds().Dy())))
		e.backgroundLayers.Add(layerCanvas)
	}
	e.backgroundLayers.Refresh()
}

// LoadBackgroundLayersOrdered loads static_images in a specific key order (not alphabetical).
func (e *EditorApp) LoadBackgroundLayersOrdered(themeDir string, order []string) {
	e.backgroundLayers.RemoveAll()

	if e.currentTheme.StaticImages == nil || len(order) == 0 {
		e.backgroundLayers.Refresh()
		return
	}

	for _, key := range order {
		imgData, ok := e.currentTheme.StaticImages[key]
		if !ok {
			continue
		}
		absPath := filepath.Join(themeDir, imgData.Path)
		log.Printf("[LoadBackgroundLayersOrdered] Layer %q: %s at (%d,%d)", key, absPath, imgData.X, imgData.Y)

		file, err := os.Open(absPath)
		if err != nil {
			log.Printf("[LoadBackgroundLayersOrdered] ERRO ao abrir %s: %v", absPath, err)
			continue
		}

		img, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			log.Printf("[LoadBackgroundLayersOrdered] ERRO ao decodificar %s: %v", absPath, err)
			continue
		}

		layerCanvas := canvas.NewImageFromImage(img)
		layerCanvas.FillMode = canvas.ImageFillOriginal
		layerCanvas.Move(fyne.NewPos(float32(imgData.X), float32(imgData.Y)))
		layerCanvas.Resize(fyne.NewSize(float32(img.Bounds().Dx()), float32(img.Bounds().Dy())))
		e.backgroundLayers.Add(layerCanvas)
	}
	e.backgroundLayers.Refresh()
}

func (e *EditorApp) loadVideoAsBackground(videoPath string) error {
	log.Printf("[loadVideoAsBackground] Iniciando stream: %s", videoPath)
	if err := e.videoPlayer.LoadVideo(videoPath); err != nil {
		log.Printf("[loadVideoAsBackground] ERRO: %v", err)
		return err
	}
	e.videoPath = videoPath
	log.Printf("[loadVideoAsBackground] Vídeo stream iniciado em loop")
	return nil
}

// SaveThemeDirect saves the theme to the current themePath (overwrite).
func (e *EditorApp) SaveThemeDirect() {
	if e.themePath == "" {
		e.SaveThemeAs()
		return
	}

	e.cleanThemeForSave()
	e.layersPanel.ApplyOrderToTheme()
	e.captureZOrder()

	// Preserve display
	if e.currentTheme.Display == nil {
		e.currentTheme.Display = &theme.Display{
			Size:        "5\"",
			Orientation: "landscape",
		}
	}

	// Update static texts
	if len(e.editableStaticText) > 0 {
		newStaticTextMap := make(map[string]theme.Text)
		for i, textData := range e.editableStaticText {
			key := fmt.Sprintf("custom_text_%d", i)
			newStaticTextMap[key] = *textData
		}
		e.currentTheme.StaticText = newStaticTextMap
	} else {
		e.currentTheme.StaticText = nil
	}

	err := theme.SaveTheme(e.currentTheme, e.themePath)
	if err != nil {
		dialog.ShowError(err, e.activeWindow())
		return
	}
	log.Printf("Tema salvo em: %s", e.themePath)
	dialog.ShowInformation("Salvo", fmt.Sprintf("Tema salvo em %s", filepath.Base(e.themePath)), e.activeWindow())
}

// captureZOrder saves the current canvas order as INDEX on each widget.
func (e *EditorApp) captureZOrder() {
	for i, obj := range e.canvasElements.Objects {
		switch w := obj.(type) {
		case *DraggableWidget:
			w.TextData.Index = i
		case *DraggableGraph:
			w.GraphData.Index = i
		case *DraggableRadial:
			w.RadialData.Index = i
		case *DraggableChart:
			w.ChartData.Index = i
		}
	}
}

// SaveThemeAs opens a file dialog to choose where to save.
func (e *EditorApp) SaveThemeAs() {
	fileSaveDialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, e.activeWindow())
				return
			}
			if writer == nil {
				return
			}
			uri := writer.URI()
			writer.Close()
			e.themePath = uri.Path()
			e.themeDir = filepath.Dir(e.themePath)
			e.SaveTheme(uri)
		},
		e.activeWindow(),
	)
	fileSaveDialog.SetFileName("theme.yaml")
	fileSaveDialog.Resize(fyne.NewSize(900, 600))
	fileSaveDialog.Show()
}

func (e *EditorApp) SaveTheme(uri fyne.URI) {
	if uri == nil {
		return
	}

	// Preserve original display — only update if it was never set
	if e.currentTheme.Display == nil {
		e.currentTheme.Display = &theme.Display{
			Size:        "5\"",
			Orientation: "landscape",
		}
	}

	targetDir := filepath.Dir(uri.Path())

	// Static texts from editor
	if len(e.editableStaticText) > 0 {
		newStaticTextMap := make(map[string]theme.Text)
		for i, textData := range e.editableStaticText {
			key := fmt.Sprintf("custom_text_%d", i)
			newStaticTextMap[key] = *textData
		}
		e.currentTheme.StaticText = newStaticTextMap
	} else {
		e.currentTheme.StaticText = nil
	}

	// Video themes: save video_play and keep static_images as-is
	if e.videoPath != "" {
		// Copy video file to target dir
		videoDestPath := filepath.Join(targetDir, filepath.Base(e.videoPath))
		err := utils.CopyFile(e.videoPath, videoDestPath)
		if err != nil {
			dialog.ShowError(err, e.activeWindow())
			return
		}

		// Copy background image if set
		bgFileName := ""
		if e.backgroundPath != "" {
			bgFileName = filepath.Base(e.backgroundPath)
			bgDestPath := filepath.Join(targetDir, bgFileName)
			err := utils.CopyFile(e.backgroundPath, bgDestPath)
			if err != nil {
				dialog.ShowError(err, e.activeWindow())
				return
			}
		}

		// Find existing video_play key or use "BACKGROUND_VIDEO"
		e.currentTheme.VideoPlay = &theme.DinamicImage{
			Path:       filepath.Base(e.videoPath),
			X:          0,
			Y:          0,
			Width:      800,
			Height:     480,
			Background: bgFileName,
		}
		e.currentTheme.StaticImages = nil
	} else {
		// Static theme: save static_images
		e.currentTheme.VideoPlay = nil

		// If user loaded a background manually, use that
		if e.backgroundPath != "" {
			bgDestPath := filepath.Join(targetDir, "background.png")
			err := utils.CopyFile(e.backgroundPath, bgDestPath)
			if err != nil {
				dialog.ShowError(err, e.activeWindow())
				return
			}
			if e.currentTheme.StaticImages == nil {
				e.currentTheme.StaticImages = make(map[string]theme.StaticImage)
			}
			e.currentTheme.StaticImages["BACKGROUND"] = theme.StaticImage{Path: "background.png", X: 0, Y: 0, Width: 800, Height: 480}
		}
		// If theme already had static_images (loaded from file), preserve them
		// and copy the referenced files to target dir
		if e.currentTheme.StaticImages != nil {
			for key, img := range e.currentTheme.StaticImages {
				srcPath := img.Path
				// If it's a relative path, try to find it from the original theme dir
				if !filepath.IsAbs(srcPath) {
					// Already relative — it will be saved as-is in YAML
					// Just make sure the file exists in target
					_ = key
				}
			}
		}
	}

	// Clean runtime fields from sensor Text structs before saving
	e.cleanThemeForSave()

	// Renumber layer keys to match display order (BACKGROUND, LAYER_2, LAYER_3...)
	e.layersPanel.ApplyOrderToTheme()

	// Save widget z-order
	e.captureZOrder()

	err := theme.SaveTheme(e.currentTheme, uri.Path())
	if err != nil {
		dialog.ShowError(err, e.activeWindow())
		return
	}
	log.Printf("Tema salvo com sucesso em: %s", uri.Path())
	dialog.ShowInformation("Sucesso", "Tema salvo com sucesso!", e.activeWindow())
}

// cleanThemeForSave removes runtime-only data from sensor Text fields
// and nils out empty structs to avoid saving them in YAML.
func (e *EditorApp) cleanThemeForSave() {
	if e.currentTheme.Stats == nil {
		return
	}

	// Helper to clean a Text field (remove runtime TEXT, keep PLACEHOLDER)
	cleanText := func(t *theme.Text) {
		if t != nil {
			t.Text = "" // TEXT field is runtime-only for sensors
		}
	}

	// Helper to clean a Measurement's child texts
	cleanMeasurement := func(m *theme.Measurement) {
		if m == nil {
			return
		}
		cleanText(m.Text)
		cleanText(m.PercentText)
	}

	// CPU
	if cpu := e.currentTheme.Stats.CPU; cpu != nil {
		cleanMeasurement(cpu.Percentage)
		cleanMeasurement(cpu.Frequency)
		cleanMeasurement(cpu.Temperature)
		cleanMeasurement(cpu.Fan)
		cleanMeasurement(cpu.Power)
		cleanMeasurement(cpu.Voltage)
	}

	// GPU
	if gpu := e.currentTheme.Stats.GPU; gpu != nil {
		cleanMeasurement(gpu.Percentage)
		cleanMeasurement(gpu.Temperature)
		cleanMeasurement(gpu.Memory)
		cleanMeasurement(gpu.Power)
		cleanMeasurement(gpu.Frequency)
		cleanMeasurement(gpu.Voltage)
		cleanMeasurement(gpu.Fan)
	}

	// Memory
	if mem := e.currentTheme.Stats.Memory; mem != nil {
		if mem.Virtual != nil {
			cleanText(mem.Virtual.PercentText)
			cleanText(mem.Virtual.Used)
			cleanText(mem.Virtual.Free)
			// Nil out if empty
			if mem.Virtual.PercentText == nil && mem.Virtual.Used == nil && mem.Virtual.Free == nil && mem.Virtual.Graph == nil && mem.Virtual.Radial == nil && mem.Virtual.Chart == nil {
				mem.Virtual = nil
			}
		}
		if mem.Swap != nil {
			cleanText(mem.Swap.PercentText)
			cleanText(mem.Swap.Used)
			cleanText(mem.Swap.Free)
			// Nil out if empty
			if mem.Swap.PercentText == nil && mem.Swap.Used == nil && mem.Swap.Free == nil && mem.Swap.Graph == nil && mem.Swap.Radial == nil && mem.Swap.Chart == nil {
				mem.Swap = nil
			}
		}
	}

	// Disk
	if disk := e.currentTheme.Stats.Disk; disk != nil {
		cleanMeasurement(disk.Used)
		cleanMeasurement(disk.Free)
		cleanMeasurement(disk.Total)
		cleanMeasurement(disk.Temperature)
	}

	// Network
	if net := e.currentTheme.Stats.Net; net != nil {
		cleanNetMeasurement := func(nm *theme.NetworkMeasurement) {
			if nm == nil {
				return
			}
			cleanMeasurement(nm.Upload)
			cleanMeasurement(nm.Download)
			cleanMeasurement(nm.Uploaded)
			cleanMeasurement(nm.Downloaded)
		}
		cleanNetMeasurement(net.Wired)
		cleanNetMeasurement(net.Wifi)

		// Nil out empty network measurements
		if net.Wifi != nil && net.Wifi.Upload == nil && net.Wifi.Download == nil && net.Wifi.Uploaded == nil && net.Wifi.Downloaded == nil {
			net.Wifi = nil
		}
		if net.Wired != nil && net.Wired.Upload == nil && net.Wired.Download == nil && net.Wired.Uploaded == nil && net.Wired.Downloaded == nil {
			net.Wired = nil
		}
	}

	// Date
	if dt := e.currentTheme.Stats.Date; dt != nil {
		cleanMeasurement(dt.Hour)
		cleanMeasurement(dt.Day)
	}

	// Weather
	if w := e.currentTheme.Stats.Weather; w != nil {
		cleanMeasurement(w.Temperature)
		cleanText(w.Condition)
	}

	// Volume
	if vol := e.currentTheme.Stats.Volume; vol != nil {
		cleanText(vol.Text)
	}
}

func (e *EditorApp) Run() {
	// Build editor layout (without bottom bar — Play/Stop are in the home)
	toolbox := buildToolbox(e)
	layersCol := e.layersPanel.container
	e.editorContent = container.NewBorder(
		nil,
		nil,
		container.NewHBox(toolbox, layersCol),
		e.propertiesPanel.container,
		buildCanvas(e),
	)

	// Build home screen
	e.homeScreen = NewHomeScreen(e)

	// System tray (only available on desktop drivers that support it)
	shown := true
	if desk, ok := e.fyneApp.(desktop.App); ok {
		if iconBytes, err := os.ReadFile("res/icon.svg"); err == nil {
			desk.SetSystemTrayIcon(fyne.NewStaticResource("icon.svg", iconBytes))
		}

		trayMenu := fyne.NewMenu("Turing Screen",
			fyne.NewMenuItem("Mostrar / Ocultar", func() {
				if shown {
					e.mainWindow.Hide()
					shown = false
				} else {
					e.mainWindow.Show()
					e.mainWindow.RequestFocus()
					shown = true
				}
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Configurações...", func() {
				if !shown {
					e.mainWindow.Show()
					e.mainWindow.RequestFocus()
					shown = true
				}
				ShowConfigDialog(e)
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Sair", func() { e.fyneApp.Quit() }),
		)
		desk.SetSystemTrayMenu(trayMenu)

		// Close intercept: hide to tray instead of quitting
		e.mainWindow.SetCloseIntercept(func() {
			shown = false
			e.mainWindow.Hide()
		})
	}

	// Main window with menu
	e.mainWindow.SetMainMenu(buildHomeMenu(e))
	e.mainWindow.SetContent(e.homeScreen.GetContainer())
	e.mainWindow.Resize(fyne.NewSize(500, 450))
	e.mainWindow.SetFixedSize(true)
	e.mainWindow.ShowAndRun()
}

// showEditor opens the theme editor in a separate window.
func (e *EditorApp) showEditor() {
	e.editorWindow = e.fyneApp.NewWindow("Turing Theme Editor")
	e.editorWindow.SetMainMenu(buildMainMenu(e))
	e.editorWindow.SetContent(e.editorContent)
	e.editorWindow.Resize(fyne.NewSize(1600, 900))
	e.editorWindow.Show()
}

// showHome closes the editor window.
func (e *EditorApp) showHome() {
	if e.editorWindow != nil {
		e.editorWindow.Close()
		e.editorWindow = nil
	}
	e.homeScreen.loadThemeList()
	e.homeScreen.updateDisplay()
}

// currentActiveTheme reads the theme name from config.yaml.
func (e *EditorApp) currentActiveTheme() string {
	// Read from config file directly
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile("conf/config.yaml")
	if err := v.ReadInConfig(); err != nil {
		return ""
	}
	return v.GetString("device.theme")
}

// SendThemeToDevice composites the current theme into a single 800x480 image
// and sends it to the turing-screen service via gRPC.
