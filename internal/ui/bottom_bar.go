package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

// buildBottomBar creates the bottom status bar with Play/Stop/Device buttons.
func buildBottomBar(app *EditorApp) fyne.CanvasObject {
	bgColor := color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}
	bg := canvas.NewRectangle(bgColor)

	playBtn := widget.NewButton("Play", func() {
		app.SendThemeToDevice()
	})

	stopBtn := widget.NewButton("Stop", func() {
		// TODO: call turing-screen Stop via gRPC
		dialog.ShowInformation("Stop", "Stop via gRPC ainda não implementado.", app.mainWindow)
	})

	deviceBtn := widget.NewButton("Device", func() {
		ShowDeviceDialog(app)
	})

	app.statusLabel = widget.NewLabel("Desconectado")

	buttons := container.NewHBox(playBtn, stopBtn, deviceBtn)

	return container.NewStack(
		bg,
		container.NewHBox(
			buttons,
			layout.NewSpacer(),
			app.statusLabel,
		),
	)
}

// updateDeviceStatus updates the status label in the bottom bar.
func (e *EditorApp) updateDeviceStatus() {
	if e.statusLabel == nil {
		return
	}
	// TODO: check gRPC connection status to turing-screen
	e.statusLabel.SetText("Desconectado")
}

// ensureDeviceConnected attempts to connect to turing-screen via gRPC.
func (e *EditorApp) ensureDeviceConnected() error {
	// TODO: implement gRPC connection to turing-screen
	return fmt.Errorf("conexão gRPC com turing-screen ainda não implementada")
}

// cleanupDevice disconnects from turing-screen.
func (e *EditorApp) cleanupDevice() {
	// TODO: disconnect gRPC client
	e.updateDeviceStatus()
}
