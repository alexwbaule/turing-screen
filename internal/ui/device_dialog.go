package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// DeviceDialog manages the device management dialog.
type DeviceDialog struct {
	app          *EditorApp
	win          fyne.Window
	storageLabel *widget.Label
	fileList     *widget.List
	files        []string
	selectedFile int
	statusLabel  *widget.Label
	dirSelect    *widget.Select
	currentDir   string
}

// ShowDeviceDialog creates and shows the device management dialog.
func ShowDeviceDialog(app *EditorApp) {
	dd := &DeviceDialog{
		app:          app,
		selectedFile: -1,
		files:        []string{},
		currentDir:   "/root/video/",
	}

	dd.win = app.mainWindow
	dd.statusLabel = widget.NewLabel("Verificando...")
	dd.storageLabel = widget.NewLabel("Storage: --")

	// Directory selector
	dirs := []string{"/root/video/", "/root/image/", "/root/font/", "/root/"}
	dd.dirSelect = widget.NewSelect(dirs, func(s string) {
		dd.currentDir = s
	})
	dd.dirSelect.SetSelected(dd.currentDir)

	dd.fileList = &widget.List{
		Length: func() int { return len(dd.files) },
		CreateItem: func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		UpdateItem: func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < len(dd.files) {
				obj.(*widget.Label).SetText(dd.files[i])
			}
		},
		OnSelected: func(id widget.ListItemID) {
			dd.selectedFile = id
		},
	}

	refreshStorageBtn := widget.NewButton("Refresh Storage", func() {
		go dd.refreshStorage()
	})

	refreshFilesBtn := widget.NewButton("Refresh Arquivos", func() {
		go dd.refreshFiles()
	})

	uploadBtn := widget.NewButton("Upload", func() {
		dd.uploadFile()
	})

	deleteBtn := widget.NewButton("Deletar", func() {
		go dd.deleteFile()
	})

	playBtn := widget.NewButton("Play Video", func() {
		go dd.playSelected()
	})

	stopBtn := widget.NewButton("Stop Video", func() {
		go dd.stopPlayback()
	})

	leftPanel := container.NewVBox(
		widget.NewLabelWithStyle("Storage", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dd.storageLabel,
		refreshStorageBtn,
		widget.NewSeparator(),
		dd.statusLabel,
	)

	// Right panel: file list with actions as column on the right
	fileActions := container.NewVBox(
		refreshFilesBtn,
		widget.NewSeparator(),
		uploadBtn,
		deleteBtn,
		widget.NewSeparator(),
		playBtn,
		stopBtn,
	)

	rightPanel := container.NewBorder(
		dd.dirSelect,
		nil,
		nil,
		fileActions,
		dd.fileList,
	)

	content := container.NewBorder(nil, nil, leftPanel, nil, rightPanel)

	d := dialog.NewCustom("Device Storage", "Fechar", content, dd.win)
	d.Resize(fyne.NewSize(700, 400))
	d.Show()

	// Load data
	go dd.refresh()
}

func (dd *DeviceDialog) refresh() {
	dd.refreshStorage()
}

func (dd *DeviceDialog) refreshStorage() {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(daemonURL + "/storage/info")
	if err != nil {
		fyne.Do(func() {
			dd.statusLabel.SetText("⚫ Não conectado")
			dd.storageLabel.SetText("Storage: indisponível")
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		fyne.Do(func() {
			dd.storageLabel.SetText("Storage: " + errResp.Error)
			dd.statusLabel.SetText("🟢 Conectado")
		})
		return
	}

	var info struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
		Free  int64 `json:"free"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		fyne.Do(func() { dd.storageLabel.SetText("Storage: erro ao ler") })
		return
	}

	fyne.Do(func() {
		dd.storageLabel.SetText(fmt.Sprintf("Total: %s | Usado: %s | Livre: %s",
			formatBytes(info.Total), formatBytes(info.Used), formatBytes(info.Free)))
		dd.statusLabel.SetText("🟢 Conectado")
	})
}

func (dd *DeviceDialog) refreshFiles() {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(daemonURL + "/storage/files?path=" + dd.currentDir)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var result struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	fyne.Do(func() {
		dd.files = result.Files
		dd.selectedFile = -1
		dd.fileList.Refresh()
	})
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

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		fyne.Do(func() { dd.statusLabel.SetText("Erro ao criar form") })
		return
	}
	part.Write(data)
	writer.Close()

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(daemonURL+"/storage/upload", writer.FormDataContentType(), &buf)
	if err != nil {
		fyne.Do(func() { dd.statusLabel.SetText("Erro ao enviar: " + err.Error()) })
		return
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fyne.Do(func() { dd.statusLabel.SetText("✓ " + filename + " enviado") })
		dd.refreshFiles()
	} else {
		fyne.Do(func() { dd.statusLabel.SetText("Erro ao enviar") })
	}
}

func (dd *DeviceDialog) deleteFile() {
	if dd.selectedFile < 0 || dd.selectedFile >= len(dd.files) {
		return
	}
	fileName := dd.files[dd.selectedFile]
	fullPath := dd.currentDir + fileName

	client := &http.Client{Timeout: 5 * time.Second}
	body := fmt.Sprintf(`{"path":"%s"}`, fullPath)
	req, _ := http.NewRequest("DELETE", daemonURL+"/storage/file", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fyne.Do(func() { dd.statusLabel.SetText("Erro: " + err.Error()) })
		return
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fyne.Do(func() { dd.statusLabel.SetText("✓ Deletado: " + fileName + " (clique Refresh)") })
	} else {
		fyne.Do(func() { dd.statusLabel.SetText("Erro ao deletar") })
	}
}

func (dd *DeviceDialog) playSelected() {
	if dd.selectedFile < 0 || dd.selectedFile >= len(dd.files) {
		return
	}
	fileName := dd.files[dd.selectedFile]
	fullPath := dd.currentDir + fileName

	client := &http.Client{Timeout: 5 * time.Second}
	body := fmt.Sprintf(`{"path":"%s"}`, fullPath)
	resp, err := client.Post(daemonURL+"/theme/video/start", "application/json", strings.NewReader(body))
	if err != nil {
		fyne.Do(func() { dd.statusLabel.SetText("Erro: " + err.Error()) })
		return
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fyne.Do(func() { dd.statusLabel.SetText("▶ Playing: " + fileName) })
	} else {
		fyne.Do(func() { dd.statusLabel.SetText("Erro ao reproduzir") })
	}
}

func (dd *DeviceDialog) stopPlayback() {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(daemonURL+"/theme/video/stop", "application/json", nil)
	if err != nil {
		fyne.Do(func() { dd.statusLabel.SetText("Erro: " + err.Error()) })
		return
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fyne.Do(func() { dd.statusLabel.SetText("⏹ Vídeo parado") })
	} else {
		fyne.Do(func() { dd.statusLabel.SetText("Erro ao parar") })
	}
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
