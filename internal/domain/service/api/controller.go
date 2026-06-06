package api

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
)

// DaemonController implements the Controller interface for the turing-screen daemon.
type DaemonController struct {
	log       *logger.Logger
	mu        sync.Mutex
	mode      string // "normal" or "editor"
	themeName string
	firmware  string
	startTime time.Time

	// Serial and commands
	serial     serial.SerialSender
	cmdDevice  *command.Device
	cmdBright  *command.Brightness
	cmdPayload *command.Payload

	// Sensor control
	stopFunc    func()
	restartFunc func() // function to restart sensors with current theme

	// Theme
	themesDir string
}

func NewDaemonController(
	log *logger.Logger,
	ser serial.SerialSender,
	cmdDevice *command.Device,
	cmdBright *command.Brightness,
	cmdPayload *command.Payload,
	themeName string,
	firmware string,
) *DaemonController {
	return &DaemonController{
		log:        log,
		mode:       "normal",
		themeName:  themeName,
		firmware:   firmware,
		startTime:  time.Now(),
		serial:     ser,
		cmdDevice:  cmdDevice,
		cmdBright:  cmdBright,
		cmdPayload: cmdPayload,
		themesDir:  "res/themes",
	}
}

// SetSensorControl sets the stop/start functions for sensor lifecycle.
func (dc *DaemonController) SetSensorControl(stop func(), start func()) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.stopFunc = stop
	dc.restartFunc = start
}

func (dc *DaemonController) GetStatus() StatusResponse {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return StatusResponse{
		Mode:       dc.mode,
		Theme:      dc.themeName,
		Firmware:   dc.firmware,
		Uptime:     time.Since(dc.startTime).Round(time.Second).String(),
		APIVersion: "1.0",
	}
}

func (dc *DaemonController) SetModeEditor() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if dc.mode == "editor" {
		return nil
	}
	dc.mode = "editor"
	// Stop sensor goroutines
	if dc.stopFunc != nil {
		go dc.stopFunc()
	}
	dc.log.Info("API: switched to editor mode (sensors stopped)")
	return nil
}

func (dc *DaemonController) SetModeNormal() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if dc.mode == "normal" {
		return nil
	}
	dc.mode = "normal"
	// Restart sensors
	if dc.restartFunc != nil {
		go dc.restartFunc()
	}
	dc.log.Info("API: switched to normal mode (sensors resumed)")
	return nil
}

func (dc *DaemonController) SetBrightness(value int) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("brightness must be 0-100")
	}
	_, err := dc.serial.Execute(dc.cmdBright.SetBrightness(value))
	if err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}
	dc.log.Infof("API: brightness set to %d", value)
	return nil
}

func (dc *DaemonController) RestartDevice() error {
	dc.log.Info("API: restart device (soft 0x82)")
	cmdStorage := command.NewStorage(dc.log)
	dc.serial.Execute(cmdStorage.RestartDevice())
	return nil
}

func (dc *DaemonController) RebootDevice() error {
	dc.log.Info("API: reboot device (hard 0x84)")
	cmdDevice := command.NewDevice(dc.log)
	dc.serial.Execute(cmdDevice.Restart())
	return nil
}

func (dc *DaemonController) ResetUSB() error {
	dc.log.Info("API: USB reset — stopping sensors first")
	// Must stop sensors before reset (they use the serial port)
	if dc.stopFunc != nil {
		dc.stopFunc()
	}
	dc.mu.Lock()
	dc.mode = "editor"
	dc.mu.Unlock()

	err := dc.serial.ResetDevice()
	if err != nil {
		return fmt.Errorf("USB reset failed: %w", err)
	}

	dc.log.Info("API: USB reset complete, restarting sensors")
	dc.mu.Lock()
	dc.mode = "normal"
	dc.mu.Unlock()
	if dc.restartFunc != nil {
		go dc.restartFunc()
	}
	return nil
}

func (dc *DaemonController) TurnOff() error {
	_, err := dc.serial.Execute(dc.cmdDevice.TurnOff())
	return err
}

func (dc *DaemonController) PreviewImage(imgData []byte) error {
	dc.mu.Lock()
	if dc.mode != "editor" {
		dc.mu.Unlock()
		return fmt.Errorf("must be in editor mode to preview")
	}
	dc.mu.Unlock()

	img, err := png.Decode(bytes.NewReader(imgData))
	if err != nil {
		return fmt.Errorf("invalid PNG: %w", err)
	}

	// Send as static bitmap
	processedImg := device.NewImageProcess(toNRGBA(img))
	_, err = dc.serial.Execute(dc.cmdPayload.SendStaticBitmap(processedImg))
	if err != nil {
		return fmt.Errorf("failed to send preview: %w", err)
	}
	dc.log.Info("API: preview image sent to device")
	return nil
}

func (dc *DaemonController) ApplyTheme(name string) error {
	dc.mu.Lock()
	dc.themeName = name
	dc.mu.Unlock()

	// Stop sensors if running
	dc.SetModeEditor()

	// TODO: reload theme and restart — for now just update config
	// The full implementation would reload the theme from disk and re-init
	dc.log.Infof("API: theme applied: %s (restart required for full effect)", name)

	// Resume with new theme
	dc.SetModeNormal()
	return nil
}

func (dc *DaemonController) GetCurrentTheme() string {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return dc.themeName
}

func (dc *DaemonController) GetThemeList() []string {
	entries, err := os.ReadDir(dc.themesDir)
	if err != nil {
		dc.log.Warnf("API: failed to list themes: %v", err)
		return nil
	}
	var themes []string
	for _, e := range entries {
		if e.IsDir() {
			// Check if theme.yaml exists
			if _, err := os.Stat(filepath.Join(dc.themesDir, e.Name(), "theme.yaml")); err == nil {
				themes = append(themes, e.Name())
			}
		}
	}
	return themes
}

func (dc *DaemonController) GetSensorValues() map[string]interface{} {
	// TODO: implement sensor value snapshot
	return map[string]interface{}{
		"status": "not implemented yet",
	}
}

func (dc *DaemonController) GetStorageInfo() (StorageInfo, error) {
	cmdStorage := command.NewStorage(dc.log)

	// Storage commands need more time — device processes filesystem
	// Try up to 3 times with increasing delay
	var resp []byte
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = dc.serial.Execute(cmdStorage.GetStorageStatus())
		if err == nil {
			break
		}
		dc.log.Debugf("GET_STORAGE_STATUS attempt %d failed, waiting...", attempt+1)
		time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
	}
	if err != nil {
		return StorageInfo{}, fmt.Errorf("storage status failed: %w", err)
	}

	info, err := command.ParseStorageInfo(resp)
	if err != nil {
		return StorageInfo{}, err
	}
	return StorageInfo{
		Total: int64(info.TotalKB) * 1024,
		Used:  int64(info.UsedKB) * 1024,
		Free:  int64(info.FreeKB) * 1024,
	}, nil
}

func (dc *DaemonController) GetStorageFiles(path string) ([]string, error) {
	cmdStorage := command.NewStorage(dc.log)

	// Storage commands need more time — device processes filesystem
	var resp []byte
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = dc.serial.Execute(cmdStorage.ListDir(path))
		if err == nil {
			break
		}
		dc.log.Debugf("LIST_DIR attempt %d failed, waiting...", attempt+1)
		time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
	}
	if err != nil {
		return nil, fmt.Errorf("list dir failed: %w", err)
	}

	// Parse "result:dir:file:name1/name2/..."
	raw := strings.TrimRight(string(bytes.Trim(resp, "\x00")), "/")
	if idx := strings.Index(raw, "file:"); idx >= 0 {
		raw = raw[idx+5:]
	} else if idx := strings.Index(raw, "dir:"); idx >= 0 {
		raw = raw[idx+4:]
	}
	if raw == "" {
		return []string{}, nil
	}
	files := strings.Split(raw, "/")
	return files, nil
}

func (dc *DaemonController) UploadFile(name string, data []byte) error {
	// TODO: implement full upload flow (CreateFile + send chunks)
	return fmt.Errorf("upload not yet implemented")
}

func (dc *DaemonController) DeleteFile(path string) error {
	cmdStorage := command.NewStorage(dc.log)
	dc.serial.Execute(cmdStorage.DeleteFile(path))
	// Device needs significant time after delete
	time.Sleep(2 * time.Second)
	return nil
}

func (dc *DaemonController) PlayVideo(path string) error {
	// Stop any current playback first
	cmdMedia := command.NewMedia(dc.log)
	dc.serial.Execute(cmdMedia.StopVideo())
	time.Sleep(200 * time.Millisecond)

	cmdStorage := command.NewStorage(dc.log)
	_, err := dc.serial.Execute(cmdStorage.PlayVideo(path, true))
	if err != nil {
		return fmt.Errorf("play video failed: %w", err)
	}
	dc.log.Infof("API: playing video %s", path)
	return nil
}

func (dc *DaemonController) StopVideo() error {
	cmdMedia := command.NewMedia(dc.log)
	dc.serial.Execute(cmdMedia.StopVideo())
	dc.log.Info("API: video stopped")
	return nil
}

// --- helpers ---

func toNRGBA(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	dst := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}
	return dst
}
