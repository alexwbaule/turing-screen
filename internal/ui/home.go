package ui

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
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
	hs := &HomeScreen{app: app, currentIndex: 0}
	hs.loadThemeList()

	activeTheme := app.currentActiveTheme()
	for i, t := range hs.themes {
		if t == activeTheme {
			hs.currentIndex = i
			break
		}
	}

	hs.previewImage = canvas.NewImageFromImage(nil)
	hs.previewImage.FillMode = canvas.ImageFillContain
	hs.previewImage.SetMinSize(fyne.NewSize(400, 240))
	hs.themeLabel = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	prevBtn := widget.NewButtonWithIcon("", fyneTheme.NavigateBackIcon(), func() { hs.navigate(-1) })
	nextBtn := widget.NewButtonWithIcon("", fyneTheme.NavigateNextIcon(), func() { hs.navigate(1) })
	hs.activateBtn = widget.NewButton("Ativar", func() { hs.applyTheme() })
	hs.editBtn = widget.NewButton("Editar Tema", func() { hs.openEditor() })
	hs.playBtn = widget.NewButtonWithIcon("Play", fyneTheme.MediaPlayIcon(), func() { hs.playTheme() })
	hs.stopBtn = widget.NewButtonWithIcon("Stop", fyneTheme.MediaStopIcon(), func() { hs.stopTheme() })
	hs.storageBtn = widget.NewButtonWithIcon("Storage", fyneTheme.StorageIcon(), func() { hs.showStorage() })
	hs.statusLabel = widget.NewLabel("Verificando...")

	go hs.pollStatus()

	navRow := container.NewBorder(nil, nil, prevBtn, nextBtn, container.NewCenter(hs.themeLabel))
	previewContainer := container.NewCenter(hs.previewImage)
	themeActions := container.NewHBox(layout.NewSpacer(), hs.activateBtn, hs.editBtn, layout.NewSpacer())
	footer := container.NewBorder(nil, nil,
		container.NewHBox(hs.playBtn, hs.stopBtn, hs.storageBtn),
		hs.statusLabel,
	)

	hs.container = container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), footer),
		nil, nil,
		container.NewVBox(previewContainer, navRow, themeActions),
	)
	hs.updateDisplay()
	return hs
}

func buildHomeMenu(app *EditorApp) *fyne.MainMenu {
	hideItem := fyne.NewMenuItem("Sair", func() {
		app.shown = false
		app.mainWindow.Hide()
	})
	quitItem := fyne.NewMenuItem("Finalizar", func() { app.fyneApp.Quit() })
	quitOffItem := fyne.NewMenuItem("Finalizar e Desligar", func() {
		if app.wsClient != nil && app.wsClient.IsConnected() {
			app.wsClient.SendFire("device.turnoff", nil)
		}
		app.fyneApp.Quit()
	})
	fileMenu := fyne.NewMenu("Arquivo", hideItem, fyne.NewMenuItemSeparator(), quitItem, quitOffItem)

	restartItem := fyne.NewMenuItem("Reiniciar", func() { go wsDeviceCmd(app, "device.restart", "Reiniciando...") })
	rebootItem := fyne.NewMenuItem("Reboot", func() { go wsDeviceCmd(app, "device.reboot", "Rebooting...") })
	resetItem := fyne.NewMenuItem("Reset USB", func() { go wsDeviceCmd(app, "device.reset", "Resetando USB...") })
	turnoffItem := fyne.NewMenuItem("Desligar", func() { go wsDeviceCmd(app, "device.turnoff", "Desligando...") })
	deviceMenu := fyne.NewMenu("Device", restartItem, rebootItem, resetItem, fyne.NewMenuItemSeparator(), turnoffItem)

	configItem := fyne.NewMenuItem("Configurações...", func() { ShowConfigDialog(app) })
	configMenu := fyne.NewMenu("Configurações", configItem)

	return fyne.NewMainMenu(fileMenu, deviceMenu, configMenu)
}

func wsDeviceCmd(app *EditorApp, action, pendingMsg string) {
	if app.wsClient == nil || !app.wsClient.IsConnected() {
		return
	}
	if app.homeScreen != nil {
		fyne.Do(func() { app.homeScreen.statusLabel.SetText("⏳ " + pendingMsg) })
	}
	resp, err := app.wsClient.Send(action, nil)
	if app.homeScreen == nil {
		return
	}
	fyne.Do(func() {
		if err != nil {
			app.homeScreen.statusLabel.SetText("⚠ Erro: " + err.Error())
			return
		}
		_ = resp
		done := strings.TrimSuffix(pendingMsg, "...")
		app.homeScreen.statusLabel.SetText("✓ " + done)
	})
}

func (hs *HomeScreen) loadThemeList() {
	themesDir := "res/themes"
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return
	}
	hs.themes = nil
	for _, e := range entries {
		if e.IsDir() {
			if _, err := os.Stat(filepath.Join(themesDir, e.Name(), "theme.yaml")); err == nil {
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

	// Immediately update status label (don't wait for next event.status from daemon)
	hs.statusLabel.SetText(fmt.Sprintf("⏳ Aplicando: %s...", themeName))
	hs.updateDisplay()

	// Tell the daemon to reload the theme now
	if hs.app.wsClient != nil && hs.app.wsClient.IsConnected() {
		go func() {
			hs.app.wsClient.Send("theme.apply", map[string]string{"name": themeName})
		}()
	}
}

func (hs *HomeScreen) playTheme() {
	if hs.app.wsClient != nil && hs.app.wsClient.IsConnected() {
		hs.app.wsClient.Send("mode.normal", nil)
	}
}

func (hs *HomeScreen) stopTheme() {
	if hs.app.wsClient != nil && hs.app.wsClient.IsConnected() {
		hs.app.wsClient.Send("mode.editor", nil)
	}
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

// --- WebSocket connection ---

const daemonWSURL = "ws://localhost:9120/ws"

func (hs *HomeScreen) pollStatus() {
	hs.app.wsClient = NewWSClient(daemonWSURL)
	hs.app.wsClient.SetEventHandler(func(msg WSMessage) {
		if msg.Action == "event.status" {
			var status struct {
				Mode       string `json:"mode"`
				Theme      string `json:"theme"`
				Uptime     string `json:"uptime"`
				DeviceType string `json:"device_type"`
			}
			json.Unmarshal(msg.Payload, &status)
			fyne.Do(func() {
				hs.app.deviceType = status.DeviceType
				hs.updateFromStatus(status.Mode, status.Theme, status.Uptime)
			})
		}
	})

	for {
		if err := hs.app.wsClient.Connect(); err != nil {
			fyne.Do(func() {
				hs.statusLabel.SetText("⚫ Desconectado")
				hs.setButtonsDisconnected()
			})
			time.Sleep(3 * time.Second)
			continue
		}
		fyne.Do(func() { hs.statusLabel.SetText("🟢 Conectado") })
		for hs.app.wsClient.IsConnected() {
			time.Sleep(1 * time.Second)
		}
		fyne.Do(func() {
			hs.statusLabel.SetText("⚫ Desconectado")
			hs.setButtonsDisconnected()
		})
		time.Sleep(3 * time.Second)
	}
}

func (hs *HomeScreen) updateFromStatus(mode, theme, uptime string) {
	switch mode {
	case "editor":
		hs.statusLabel.SetText(fmt.Sprintf("🟢 ⏸ %s | %s", theme, uptime))
		hs.setButtonsPaused()
	case "starting":
		hs.statusLabel.SetText(fmt.Sprintf("🟢 ⏳ Aguardando device... | %s", uptime))
		hs.setButtonsStarting()
	default: // "normal"
		hs.statusLabel.SetText(fmt.Sprintf("🟢 ▶ %s | %s", theme, uptime))
		hs.setButtonsRunning()
	}
}

func (hs *HomeScreen) setButtonsRunning() {
	hs.playBtn.Disable()
	hs.stopBtn.Enable()
	hs.storageBtn.Enable()
}

func (hs *HomeScreen) setButtonsPaused() {
	hs.playBtn.Enable()
	hs.stopBtn.Disable()
	hs.storageBtn.Enable()
}

func (hs *HomeScreen) setButtonsStarting() {
	hs.playBtn.Disable()
	hs.stopBtn.Disable()
	hs.storageBtn.Disable()
}

func (hs *HomeScreen) setButtonsDisconnected() {
	hs.playBtn.Disable()
	hs.stopBtn.Disable()
	hs.storageBtn.Disable()
}
