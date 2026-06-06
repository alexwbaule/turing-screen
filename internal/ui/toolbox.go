package ui

import (
	"fmt"
	"image"
	_ "image/png"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/alexwbaule/turing-screen/internal/theme"
)

func buildToolbox(app *EditorApp) fyne.CanvasObject {
	var items []fyne.CanvasObject

	addSection := func(title string, buttons []struct {
		Label string
		Type  theme.ComponentType
	}) {
		items = append(items, widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		for _, b := range buttons {
			bt := b
			items = append(items, widget.NewButton(bt.Label, func() { app.AddComponent(bt.Type) }))
		}
	}

	addSection("Texto Estático", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Texto Estático", theme.STATIC_TEXT},
	})

	addSection("CPU", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Percentage (Graph)", theme.CPU_PERCENTAGE_GRAPH},
		{"Percentage (Text)", theme.CPU_PERCENTAGE_TEXT},
		{"Percentage (Radial)", theme.CPU_PERCENTAGE_RADIAL},
		{"Percentage (Chart)", theme.CPU_PERCENTAGE_CHART},
		{"Temperature", theme.CPU_TEMPERATURE_TEXT},
		{"Frequency", theme.CPU_FREQUENCY_TEXT},
		{"Fan", theme.CPU_FAN_TEXT},
		{"Power", theme.CPU_POWER_TEXT},
		{"Voltage", theme.CPU_VOLTAGE_TEXT},
		{"Load 1min", theme.CPU_LOAD_ONE_TEXT},
		{"Load 5min", theme.CPU_LOAD_FIVE_TEXT},
		{"Load 15min", theme.CPU_LOAD_FIFTEEN_TEXT},
	})

	addSection("GPU", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Percentage", theme.GPU_PERCENTAGE_GRAPH},
		{"Temperature", theme.GPU_TEMPERATURE_TEXT},
		{"Memory", theme.GPU_MEMORY_TEXT},
		{"Power", theme.GPU_POWER_TEXT},
		{"Frequency", theme.GPU_FREQUENCY_TEXT},
		{"Voltage", theme.GPU_VOLTAGE_TEXT},
		{"Fan", theme.GPU_FAN_TEXT},
	})

	addSection("Memória Virtual", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Percent Text", theme.MEMORY_VIRTUAL_PERCENT_TEXT},
		{"Graph", theme.MEMORY_VIRTUAL_GRAPH},
		{"Radial", theme.MEMORY_VIRTUAL_RADIAL},
		{"Chart", theme.MEMORY_VIRTUAL_CHART},
		{"Used", theme.MEMORY_VIRTUAL_USED_TEXT},
		{"Free", theme.MEMORY_VIRTUAL_FREE_TEXT},
	})

	addSection("Swap", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Percent Text", theme.MEMORY_SWAP_PERCENT_TEXT},
		{"Graph", theme.MEMORY_SWAP_GRAPH},
		{"Radial", theme.MEMORY_SWAP_RADIAL},
		{"Chart", theme.MEMORY_SWAP_CHART},
	})

	addSection("Disco", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Used %", theme.DISK_USED_PERCENT_TEXT},
		{"Free", theme.DISK_FREE_TEXT},
		{"Total", theme.DISK_TOTAL_TEXT},
		{"Temperature", theme.DISK_TEMPERATURE_TEXT},
	})

	addSection("Rede Ethernet", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Upload", theme.NET_ETH_UPLOAD_TEXT},
		{"Download", theme.NET_ETH_DOWNLOAD_TEXT},
		{"Total Uploaded", theme.NET_ETH_UPLOADED_TEXT},
		{"Total Downloaded", theme.NET_ETH_DOWNLOADED_TEXT},
	})

	addSection("Rede WiFi", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Upload", theme.NET_WLAN_UPLOAD_TEXT},
		{"Download", theme.NET_WLAN_DOWNLOAD_TEXT},
		{"Total Uploaded", theme.NET_WLAN_UPLOADED_TEXT},
		{"Total Downloaded", theme.NET_WLAN_DOWNLOADED_TEXT},
	})

	addSection("Data / Hora", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Hora", theme.TIME_HOUR_TEXT},
		{"Dia", theme.TIME_DAY_TEXT},
	})

	addSection("Weather", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Temperature", theme.WEATHER_TEMPERATURE_TEXT},
		{"Condition", theme.WEATHER_CONDITION_TEXT},
	})

	addSection("Volume", []struct {
		Label string
		Type  theme.ComponentType
	}{
		{"Volume", theme.VOLUME_TEXT},
	})

	scroll := container.NewVScroll(container.NewVBox(items...))
	scroll.SetMinSize(fyne.NewSize(180, 1))
	return scroll
}

// buildMainMenu creates the application main menu (Arquivo, Editar).
func buildMainMenu(app *EditorApp) *fyne.MainMenu {
	// --- Arquivo ---
	newItem := fyne.NewMenuItem("Novo", func() {
		app.NewTheme()
	})
	openItem := fyne.NewMenuItem("Abrir Tema...", func() {
		fileOpenDialog := dialog.NewFileOpen(
			func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, app.activeWindow())
					return
				}
				if reader == nil {
					return
				}
				app.LoadTheme(reader.URI().Path())
				reader.Close()
			},
			app.activeWindow(),
		)
		fileOpenDialog.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".yml"}))
		fileOpenDialog.Resize(fyne.NewSize(900, 600))
		fileOpenDialog.Show()
	})
	saveItem := fyne.NewMenuItem("Salvar", func() {
		app.SaveThemeDirect()
	})
	saveAsItem := fyne.NewMenuItem("Salvar Como...", func() {
		app.SaveThemeAs()
	})
	closeItem := fyne.NewMenuItem("Fechar Tema", func() {
		app.NewTheme()
	})
	closeEditorItem := fyne.NewMenuItem("Fechar Editor", func() {
		app.showHome()
	})
	fileMenu := fyne.NewMenu("Arquivo", newItem, openItem, fyne.NewMenuItemSeparator(), saveItem, saveAsItem, fyne.NewMenuItemSeparator(), closeItem, closeEditorItem)

	// --- Editar ---
	bgImageItem := fyne.NewMenuItem("Adicionar Imagem", func() {
		fileOpenDialog := dialog.NewFileOpen(
			func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, app.activeWindow())
					return
				}
				if reader == nil {
					return
				}
				path := reader.URI().Path()
				reader.Close()
				file, err := os.Open(path)
				if err != nil {
					dialog.ShowError(err, app.activeWindow())
					return
				}
				defer file.Close()
				config, _, err := image.DecodeConfig(file)
				if err != nil {
					dialog.ShowError(err, app.activeWindow())
					return
				}
				isValid := (config.Width == 800 && config.Height == 480) || (config.Width == 480 && config.Height == 800)
				if !isValid {
					dialog.ShowError(fmt.Errorf("resolução da imagem é %dx%d. A resolução deve ser 800x480 ou 480x800", config.Width, config.Height), app.activeWindow())
					return
				}
				app.LoadBackgroundImage(path)
			},
			app.activeWindow(),
		)
		fileOpenDialog.SetFilter(storage.NewExtensionFileFilter([]string{".png"}))
		fileOpenDialog.Resize(fyne.NewSize(900, 600))
		fileOpenDialog.Show()
	})
	bgVideoItem := fyne.NewMenuItem("Adicionar Vídeo", func() {
		if err := CheckFFmpegAvailable(); err != nil {
			dialog.ShowError(err, app.activeWindow())
			return
		}
		fileOpenDialog := dialog.NewFileOpen(
			func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, app.activeWindow())
					return
				}
				if reader == nil {
					return
				}
				path := reader.URI().Path()
				reader.Close()

				info, err := os.Stat(path)
				if err != nil {
					dialog.ShowError(err, app.activeWindow())
					return
				}
				if info.Size() > 10*1024*1024 {
					dialog.ShowError(fmt.Errorf("vídeo muito grande (%d MB). Limite: 10 MB", info.Size()/(1024*1024)), app.activeWindow())
					return
				}

				if err := app.videoPlayer.LoadVideo(path); err != nil {
					dialog.ShowError(err, app.activeWindow())
					return
				}
				app.videoPath = path
			},
			app.activeWindow(),
		)
		fileOpenDialog.SetFilter(storage.NewExtensionFileFilter([]string{".mp4"}))
		fileOpenDialog.Resize(fyne.NewSize(900, 600))
		fileOpenDialog.Show()
	})

	bgMenu := fyne.NewMenuItem("Background", nil)
	bgMenu.ChildMenu = fyne.NewMenu("", bgImageItem, bgVideoItem)

	deleteItem := fyne.NewMenuItem("Excluir Componente", func() {
		app.DeleteSelectedWidget()
	})
	editMenu := fyne.NewMenu("Editar", bgMenu, fyne.NewMenuItemSeparator(), deleteItem)

	return fyne.NewMainMenu(fileMenu, editMenu)
}
