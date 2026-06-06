package ui

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/spf13/viper"
)

// HomeScreen shows the launcher/manager.
type HomeScreen struct {
	app          *EditorApp
	container    *fyne.Container
	themes       []string
	currentIndex int
	themeLabel   *widget.Label
	previewImage *canvas.Image
	statusLabel  *widget.Label
	playBtn      *widget.Button
	stopBtn      *widget.Button
	storageBtn   *widget.Button
	activateBtn  *widget.Button
	editBtn      *widget.Button
}

func NewHomeScreen(app *EditorApp) *HomeScreen {
	hs := &HomeScreen{
		app:          app,
		currentIndex: 0,
	}

	hs.loadThemeList()

	// Find active theme
	activeTheme := app.currentActiveTheme()
	for i, t := range hs.themes {
		if t == activeTheme {
			hs.currentIndex = i
			break
		}
	}

	// Preview image
	hs.previewImage = canvas.NewImageFromImage(nil)
	hs.previewImage.FillMode = canvas.ImageFillContain
	hs.previewImage.SetMinSize(fyne.NewSize(400, 240))

	// Theme name
	hs.themeLabel = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Navigation
	prevBtn := widget.NewButtonWithIcon("", fyneTheme.NavigateBackIcon(), func() {
		hs.navigate(-1)
	})
	nextBtn := widget.NewButtonWithIcon("", fyneTheme.NavigateNextIcon(), func() {
		hs.navigate(1)
	})

	// Activate / Edit buttons
	hs.activateBtn = widget.NewButton("Ativar", func() {
		hs.applyTheme()
	})
	hs.editBtn = widget.NewButton("Editar Tema", func() {
		hs.openEditor()
	})

	// Play / Stop / Storage
	hs.playBtn = widget.NewButtonWithIcon("Play", fyneTheme.MediaPlayIcon(), func() {
		hs.playTheme()
	})
	hs.stopBtn = widget.NewButtonWithIcon("Stop", fyneTheme.MediaStopIcon(), func() {
		hs.stopTheme()
	})
	hs.storageBtn = widget.NewButtonWithIcon("Storage", fyneTheme.StorageIcon(), func() {
		hs.showStorage()
	})

	// Status
	hs.statusLabel = widget.NewLabel("Verificando...")

	// Start polling
	go hs.pollStatus()

	// --- Layout ---
	previewContainer := container.NewCenter(hs.previewImage)

	navRow := container.NewHBox(
		layout.NewSpacer(),
		prevBtn,
		hs.themeLabel,
		nextBtn,
		layout.NewSpacer(),
	)

	actionRow := container.NewHBox(
		layout.NewSpacer(),
		hs.activateBtn, hs.editBtn,
		layout.NewSpacer(),
	)

	footer := container.NewBorder(
		nil, nil,
		container.NewHBox(hs.playBtn, hs.stopBtn, hs.storageBtn),
		hs.statusLabel,
	)

	hs.container = container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), footer),
		nil, nil,
		container.NewVBox(
			previewContainer,
			navRow,
			actionRow,
		),
	)

	hs.updateDisplay()
	return hs
}

// buildHomeMenu creates the menu bar for the home window.
func buildHomeMenu(app *EditorApp) *fyne.MainMenu {
	// Arquivo
	quitOffItem := fyne.NewMenuItem("Sair e Desligar", func() {
		go func() {
			client := &http.Client{Timeout: 5 * time.Second}
			client.Post(daemonURL+"/device/turnoff", "application/json", nil)
		}()
		app.fyneApp.Quit()
	})
	quitItem := fyne.NewMenuItem("Sair", func() {
		app.fyneApp.Quit()
	})
	fileMenu := fyne.NewMenu("Arquivo", quitOffItem, quitItem)

	// Device
	restartItem := fyne.NewMenuItem("Reiniciar (soft)", func() {
		go deviceCommand("/device/restart")
	})
	rebootItem := fyne.NewMenuItem("Reboot (hard)", func() {
		go deviceCommand("/device/reboot")
	})
	resetItem := fyne.NewMenuItem("Reset USB", func() {
		go deviceCommand("/device/reset")
	})
	turnoffItem := fyne.NewMenuItem("Desligar", func() {
		go deviceCommand("/device/turnoff")
	})
	deviceMenu := fyne.NewMenu("Device", restartItem, rebootItem, resetItem, fyne.NewMenuItemSeparator(), turnoffItem)

	return fyne.NewMainMenu(fileMenu, deviceMenu)
}

func deviceCommand(endpoint string) {
	client := &http.Client{Timeout: 10 * time.Second}
	client.Post(daemonURL+endpoint, "application/json", nil)
}

func (hs *HomeScreen) loadThemeList() {
	themesDir := "res/themes"
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		log.Printf("[HomeScreen] Failed to read themes dir: %v", err)
		return
	}
	hs.themes = nil
	for _, e := range entries {
		if e.IsDir() {
			yamlPath := filepath.Join(themesDir, e.Name(), "theme.yaml")
			if _, err := os.Stat(yamlPath); err == nil {
				hs.themes = append(hs.themes, e.Name())
			}
		}
	}
	sort.Strings(hs.themes)
}

func (hs *HomeScreen) navigate(direction int) {
	if len(hs.themes) == 0 {
		return
	}
	hs.currentIndex += direction
	if hs.currentIndex < 0 {
		hs.currentIndex = len(hs.themes) - 1
	}
	if hs.currentIndex >= len(hs.themes) {
		hs.currentIndex = 0
	}
	hs.updateDisplay()
}

func (hs *HomeScreen) updateDisplay() {
	if len(hs.themes) == 0 {
		hs.themeLabel.SetText("Nenhum tema encontrado")
		return
	}

	themeName := hs.themes[hs.currentIndex]
	hs.themeLabel.SetText(fmt.Sprintf("  %s  ", themeName))

	// Load preview
	themesDir := "res/themes"
	previewPaths := []string{
		filepath.Join(themesDir, themeName, "assets", "image_0.png"),
		filepath.Join(themesDir, themeName, "background.png"),
	}

	var img image.Image
	for _, path := range previewPaths {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		decoded, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			continue
		}
		img = decoded
		break
	}

	if img != nil {
		hs.previewImage.Image = img
	} else {
		hs.previewImage.Image = image.NewRGBA(image.Rect(0, 0, 400, 240))
	}
	hs.previewImage.Refresh()

	// Update activate button state
	activeTheme := hs.app.currentActiveTheme()
	if themeName == activeTheme {
		hs.activateBtn.SetText("Ativo")
		hs.activateBtn.Disable()
	} else {
		hs.activateBtn.SetText("Ativar")
		hs.activateBtn.Enable()
	}
}

func (hs *HomeScreen) applyTheme() {
	if len(hs.themes) == 0 {
		return
	}
	themeName := hs.themes[hs.currentIndex]

	// Write to config.yaml
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile("conf/config.yaml")
	if err := v.ReadInConfig(); err != nil {
		log.Printf("[HomeScreen] Failed to read config: %v", err)
		return
	}
	v.Set("device.theme", themeName)
	if err := v.WriteConfig(); err != nil {
		log.Printf("[HomeScreen] Failed to write config: %v", err)
		return
	}

	// Tell daemon
	client := &http.Client{Timeout: 10 * time.Second}
	body := fmt.Sprintf(`{"name":"%s"}`, themeName)
	client.Post(daemonURL+"/theme/apply", "application/json", strings.NewReader(body))

	log.Printf("[HomeScreen] Theme applied: %s", themeName)
	hs.updateDisplay()
}

func (hs *HomeScreen) playTheme() {
	client := &http.Client{Timeout: 10 * time.Second}
	client.Post(daemonURL+"/mode/normal", "application/json", nil)
}

func (hs *HomeScreen) stopTheme() {
	client := &http.Client{Timeout: 5 * time.Second}
	client.Post(daemonURL+"/mode/editor", "application/json", nil)
}

func (hs *HomeScreen) openEditor() {
	if len(hs.themes) == 0 {
		return
	}
	themeName := hs.themes[hs.currentIndex]
	themePath := filepath.Join("res/themes", themeName, "theme.yaml")
	hs.app.showEditor()
	hs.app.LoadTheme(themePath)
}

func (hs *HomeScreen) showStorage() {
	ShowDeviceDialog(hs.app)
}

func (hs *HomeScreen) GetContainer() *fyne.Container {
	return hs.container
}

const daemonURL = "http://localhost:9120"

func (hs *HomeScreen) pollStatus() {
	hs.checkDaemonStatus()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		hs.checkDaemonStatus()
	}
}

func (hs *HomeScreen) checkDaemonStatus() {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(daemonURL + "/status")
	if err != nil {
		fyne.Do(func() {
			hs.statusLabel.SetText("⚫ Desconectado")
			hs.setButtonsDisconnected()
		})
		return
	}
	defer resp.Body.Close()

	var status struct {
		Mode     string `json:"mode"`
		Theme    string `json:"theme"`
		Firmware string `json:"firmware"`
		Uptime   string `json:"uptime"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		fyne.Do(func() {
			hs.statusLabel.SetText("⚫ Erro")
			hs.setButtonsDisconnected()
		})
		return
	}

	modeLabel := "▶"
	if status.Mode == "editor" {
		modeLabel = "⏸"
	}

	fyne.Do(func() {
		hs.statusLabel.SetText(fmt.Sprintf("🟢 %s %s | %s", modeLabel, status.Theme, status.Uptime))

		if status.Mode == "editor" {
			hs.setButtonsPaused()
		} else {
			hs.setButtonsRunning()
		}
	})
}

func (hs *HomeScreen) setButtonsRunning() {
	hs.playBtn.Disable()
	hs.stopBtn.Enable()
	hs.storageBtn.Disable()
}

func (hs *HomeScreen) setButtonsPaused() {
	hs.playBtn.Enable()
	hs.stopBtn.Disable()
	hs.storageBtn.Enable()
}

func (hs *HomeScreen) setButtonsDisconnected() {
	hs.playBtn.Disable()
	hs.stopBtn.Disable()
	hs.storageBtn.Disable()
}
