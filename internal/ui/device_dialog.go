package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

type DeviceDialog struct {
	app          *EditorApp
	win          fyne.Window
	totalLabel   *widget.Label
	usedLabel    *widget.Label
	freeLabel    *widget.Label
	fileList     *widget.List
	files        []string
	selectedFile int
	statusLabel  *widget.Label
	dirSelect    *widget.Select
	currentDir   string
}

func ShowDeviceDialog(app *EditorApp) {
	dd := &DeviceDialog{
		app:          app,
		selectedFile: -1,
		files:        []string{},
		currentDir:   "/root/video/",
	}

	dd.win = app.mainWindow
	dd.statusLabel = widget.NewLabel("Verificando...")
	dd.totalLabel = widget.NewLabel("Total:  --")
	dd.usedLabel = widget.NewLabel("Usado: --")
	dd.freeLabel = widget.NewLabel("Livre:  --")

	dirs := []string{"/root/video/", "/root/image/", "/root/font/", "/root/"}
	dd.dirSelect = widget.NewSelect(dirs, func(s string) {
		dd.currentDir = s
	})
	dd.dirSelect.SetSelected(dd.currentDir)

	dd.fileList = &widget.List{
		Length:     func() int { return len(dd.files) },
		CreateItem: func() fyne.CanvasObject { return widget.NewLabel("") },
		UpdateItem: func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < len(dd.files) {
				obj.(*widget.Label).SetText(dd.files[i])
			}
		},
		OnSelected: func(id widget.ListItemID) { dd.selectedFile = id },
	}

	refreshStorageBtn := widget.NewButton("Refresh Storage", func() { go dd.refreshStorage() })
	refreshFilesBtn := widget.NewButton("Refresh Arquivos", func() { go dd.refreshFiles() })
	restartBtn := widget.NewButton("Reiniciar Device", func() { go dd.restartDevice() })
	uploadBtn := widget.NewButton("Upload", func() { dd.uploadFile() })
	deleteBtn := widget.NewButton("Deletar", func() { go dd.deleteFile() })
	playBtn := widget.NewButton("Play Video", func() { go dd.playSelected() })
	stopBtn := widget.NewButton("Stop Video", func() { go dd.stopPlayback() })

	storageInfo := container.NewVBox(
		dd.totalLabel,
		dd.usedLabel,
		dd.freeLabel,
	)

	leftPanel := container.NewVBox(
		widget.NewLabelWithStyle("Storage", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		storageInfo,
		refreshStorageBtn,
		widget.NewSeparator(),
		restartBtn,
		widget.NewSeparator(),
		dd.statusLabel,
	)

	fileActions := container.NewVBox(
		refreshFilesBtn,
		widget.NewSeparator(),
		uploadBtn, deleteBtn,
		widget.NewSeparator(),
		playBtn, stopBtn,
	)

	rightPanel := container.NewBorder(dd.dirSelect, nil, nil, fileActions, dd.fileList)
	content := container.NewBorder(nil, nil, leftPanel, nil, rightPanel)

	d := dialog.NewCustom("Device Storage", "Fechar", content, dd.win)
	d.Resize(fyne.NewSize(700, 400))
	d.Show()

	go dd.refreshStorage()
}

func (dd *DeviceDialog) refreshStorage() {
	ws := dd.app.wsClient
	if ws == nil || !ws.IsConnected() {
		fyne.Do(func() {
			dd.statusLabel.SetText("⚫ Não conectado")
			dd.totalLabel.SetText("Total:  --")
			dd.usedLabel.SetText("Usado: --")
			dd.freeLabel.SetText("Livre:  --")
		})
		return
	}

	resp, err := ws.Send("storage.info", nil)
	if err != nil {
		fyne.Do(func() {
			dd.statusLabel.SetText("Erro: " + err.Error())
			dd.totalLabel.SetText("Total:  --")
			dd.usedLabel.SetText("Usado: --")
			dd.freeLabel.SetText("Livre:  --")
		})
		return
	}

	var info struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
		Free  int64 `json:"free"`
	}
	json.Unmarshal(resp.Payload, &info)

	fyne.Do(func() {
		dd.totalLabel.SetText("Total:  " + formatBytes(info.Total))
		dd.usedLabel.SetText("Usado: " + formatBytes(info.Used))
		dd.freeLabel.SetText("Livre:  " + formatBytes(info.Free))
		dd.statusLabel.SetText("🟢 Conectado")
	})
}

func (dd *DeviceDialog) refreshFiles() {
	ws := dd.app.wsClient
	if ws == nil || !ws.IsConnected() {
		return
	}

	resp, err := ws.Send("storage.files", map[string]string{"path": dd.currentDir})
	if err != nil {
		return
	}

	var result struct {
		Files []string `json:"files"`
	}
	json.Unmarshal(resp.Payload, &result)

	fyne.Do(func() {
		dd.files = result.Files
		dd.selectedFile = -1
		dd.fileList.Refresh()
	})
}

func (dd *DeviceDialog) restartDevice() {
	ws := dd.app.wsClient
	if ws == nil || !ws.IsConnected() {
		return
	}
	ws.Send("device.restart", nil)
	fyne.Do(func() { dd.statusLabel.SetText("Device reiniciado") })
}

func (dd *DeviceDialog) uploadFile() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil {
			dialog.ShowError(fmt.Errorf("erro ao ler arquivo: %w", err), dd.win)
			return
		}
		filename := reader.URI().Name()
		go dd.doUpload(filename, data)
	}, dd.win)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".mp4", ".avi", ".mkv"}))
	fileDialog.Resize(fyne.NewSize(700, 400))
	fileDialog.Show()
}

func (dd *DeviceDialog) doUpload(filename string, data []byte) {
	fyne.Do(func() { dd.statusLabel.SetText("Enviando " + filename + "...") })

	ws := dd.app.wsClient
	if ws == nil || !ws.IsConnected() {
		fyne.Do(func() { dd.statusLabel.SetText("⚫ Não conectado") })
		return
	}

	// For large files, upload via base64 (not ideal but works)
	// TODO: implement chunked upload for large files
	_, err := ws.Send("storage.upload", map[string]interface{}{
		"name": filename,
		"data": data, // will be base64 encoded by json.Marshal
	})
	if err != nil {
		fyne.Do(func() { dd.statusLabel.SetText("Erro: " + err.Error()) })
		return
	}
	fyne.Do(func() { dd.statusLabel.SetText("✓ " + filename + " enviado") })
}

func (dd *DeviceDialog) deleteFile() {
	if dd.selectedFile < 0 || dd.selectedFile >= len(dd.files) {
		return
	}
	fileName := dd.files[dd.selectedFile]
	fullPath := dd.currentDir + fileName

	ws := dd.app.wsClient
	if ws == nil || !ws.IsConnected() {
		return
	}

	_, err := ws.Send("storage.delete", map[string]string{"path": fullPath})
	if err != nil {
		fyne.Do(func() { dd.statusLabel.SetText("Erro: " + err.Error()) })
		return
	}
	fyne.Do(func() { dd.statusLabel.SetText("✓ Deletado: " + fileName + " (clique Refresh)") })

	time.Sleep(2 * time.Second)
}

func (dd *DeviceDialog) playSelected() {
	if dd.selectedFile < 0 || dd.selectedFile >= len(dd.files) {
		return
	}
	fileName := dd.files[dd.selectedFile]
	fullPath := dd.currentDir + fileName

	ws := dd.app.wsClient
	if ws == nil || !ws.IsConnected() {
		return
	}

	_, err := ws.Send("theme.video.start", map[string]string{"path": fullPath})
	if err != nil {
		fyne.Do(func() { dd.statusLabel.SetText("Erro: " + err.Error()) })
		return
	}
	fyne.Do(func() { dd.statusLabel.SetText("▶ Playing: " + fileName) })
}

func (dd *DeviceDialog) stopPlayback() {
	ws := dd.app.wsClient
	if ws == nil || !ws.IsConnected() {
		return
	}

	_, err := ws.Send("theme.video.stop", nil)
	if err != nil {
		fyne.Do(func() { dd.statusLabel.SetText("Erro: " + err.Error()) })
		return
	}
	fyne.Do(func() { dd.statusLabel.SetText("⏹ Vídeo parado") })
}

func formatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	if b < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(b)/(1024*1024*1024))
}
