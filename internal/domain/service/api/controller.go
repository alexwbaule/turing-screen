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
	mode      string // "starting", "normal", or "editor"
	themeName string
	firmware  string
	startTime time.Time
	brightness int // last brightness value sent to the device

	// Serial and commands
	serial     serial.SerialSender
	cmdDevice  *command.Device
	cmdBright  *command.Brightness
	cmdPayload *command.Payload

	// Sensor control
	stopFunc    func()
	restartFunc func() // function to restart sensors with current theme
	sensorFunc  func() map[string]interface{}

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
		mode:       "starting",
		themeName:  themeName,
		firmware:   firmware,
		startTime:  time.Now(),
		brightness: 100,
		serial:     ser,
		cmdDevice:  cmdDevice,
		cmdBright:  cmdBright,
		cmdPayload: cmdPayload,
		themesDir:  "res/themes",
	}
}

// SetSensorFunc sets the function that returns a snapshot of current sensor values.
func (dc *DaemonController) SetSensorFunc(fn func() map[string]interface{}) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.sensorFunc = fn
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
	if dc.mode == "editor" {
		dc.mu.Unlock()
		return nil
	}
	dc.mode = "editor"
	stop := dc.stopFunc
	dc.mu.Unlock()

	// Stop synchronously — caller blocks until all sensor goroutines have exited
	// and the serial buffer has been flushed. This guarantees the port is free
	// before any subsequent storage or preview operation.
	if stop != nil {
		stop()
	}
	dc.log.Info("API: switched to editor mode (sensors stopped)")
	return nil
}

func (dc *DaemonController) SetModeNormal() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if dc.mode == "normal" || dc.mode == "starting" {
		return nil
	}
	dc.mode = "starting"
	if dc.restartFunc != nil {
		go dc.restartFunc()
	}
	dc.log.Info("API: switched to starting mode (sensors resuming)")
	return nil
}

// NotifySensorsStarted is called by startSensors when sensors come online successfully.
func (dc *DaemonController) NotifySensorsStarted() {
	dc.mu.Lock()
	dc.mode = "normal"
	dc.mu.Unlock()
	dc.log.Info("API: sensors started, mode normal")
}

// NotifySensorsFailed is called by startSensors when device init fails.
func (dc *DaemonController) NotifySensorsFailed() {
	dc.mu.Lock()
	dc.mode = "editor"
	dc.mu.Unlock()
	dc.log.Warn("API: sensors failed to start, reverted to editor mode")
}

func (dc *DaemonController) SetBrightness(value int) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("brightness must be 0-100")
	}
	_, err := dc.serial.Execute(dc.cmdBright.SetBrightness(value))
	if err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}
	dc.mu.Lock()
	dc.brightness = value
	dc.mu.Unlock()
	dc.log.Infof("API: brightness set to %d", value)
	return nil
}

// requireEditor returns an error if the device is not in editor mode.
// All storage and serial-direct operations must call this first to ensure
// the sensor worker is not competing for the serial port.
func (dc *DaemonController) requireEditor() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if dc.mode != "editor" {
		return fmt.Errorf("device must be in editor mode for this operation (current mode: %s)", dc.mode)
	}
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
	// Stop sensors before turning off so the worker doesn't keep writing to the
	// serial port after the device shuts down (would trigger retry/wakeup logic).
	dc.mu.Lock()
	stop := dc.stopFunc
	dc.mode = "editor"
	dc.mu.Unlock()
	if stop != nil {
		stop()
	}
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
	// Stop sensors completely before switching theme (SetModeEditor is synchronous).
	if err := dc.SetModeEditor(); err != nil {
		return err
	}

	dc.mu.Lock()
	dc.themeName = name
	dc.mu.Unlock()

	dc.log.Infof("API: theme set to %s, restarting...", name)

	// Resume — startSensors reloads config from disk, so the caller must have
	// written the new theme name to config before calling this.
	return dc.SetModeNormal()
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
	dc.mu.Lock()
	fn := dc.sensorFunc
	dc.mu.Unlock()
	if fn != nil {
		return fn()
	}
	return map[string]interface{}{}
}

func (dc *DaemonController) GetStorageInfo() (StorageInfo, error) {
	if err := dc.requireEditor(); err != nil {
		return StorageInfo{}, err
	}
	// Flush before writing so that any trailing bytes the device sent after the
	// previous storage command are discarded. Flushing here (at the start of the
	// next command) is safer than flushing at the end of the previous one,
	// because those trailing bytes may arrive in the OS buffer after the previous
	// Flush() has already been called.
	dc.serial.Flush()

	cmdStorage := command.NewStorage(dc.log)

	var resp []byte
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			dc.serial.Flush() // clear any partial response before retry
		}
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
	if err := dc.requireEditor(); err != nil {
		return nil, err
	}
	dc.serial.Flush() // discard trailing bytes from any previous storage command

	cmdStorage := command.NewStorage(dc.log)

	var resp []byte
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			dc.serial.Flush()
		}
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
	if err := dc.requireEditor(); err != nil {
		return err
	}
	dc.serial.Flush()

	dc.mu.Lock()
	brightness := byte(dc.brightness)
	dc.mu.Unlock()

	cmdStorage := command.NewStorage(dc.log)

	// 1. Prepare device for upload
	if _, err := dc.serial.Execute(cmdStorage.SetPreUpload(brightness)); err != nil {
		return fmt.Errorf("pre-upload failed: %w", err)
	}

	// 2. Create file on device (path is always /root/video/)
	path := "/root/video/" + name
	if _, err := dc.serial.Execute(cmdStorage.CreateFile(path, int64(len(data)))); err != nil {
		return fmt.Errorf("create file failed: %w", err)
	}

	// 3. Send file data; device responds with "file_rev_done" when complete
	if _, err := dc.serial.Execute(cmdStorage.UploadFile(data)); err != nil {
		return fmt.Errorf("upload file data failed: %w", err)
	}

	dc.log.Infof("API: uploaded %s (%d bytes)", name, len(data))
	return nil
}

func (dc *DaemonController) DeleteFile(path string) error {
	if err := dc.requireEditor(); err != nil {
		return err
	}
	dc.serial.Flush()
	cmdStorage := command.NewStorage(dc.log)
	if _, err := dc.serial.Execute(cmdStorage.DeleteFile(path)); err != nil {
		// Device may not respond within timeout but the delete still happens.
		// Validate by checking the file no longer exists.
		dc.log.Warnf("DeleteFile: no response from device, verifying deletion: %v", err)
		dc.serial.Flush()
		info := cmdStorage.GetFileInfo(path)
		if resp, verr := dc.serial.Execute(info); verr == nil {
			size := string(bytes.Trim(resp, "\x00"))
			if size != "0" {
				return fmt.Errorf("delete file failed: device still reports file size %s", size)
			}
		}
	}
	dc.serial.Flush()
	dc.log.Infof("API: deleted %s", path)
	return nil
}

func (dc *DaemonController) PlayVideo(path string) error {
	if err := dc.requireEditor(); err != nil {
		return err
	}
	dc.serial.Flush()
	cmdMedia := command.NewMedia(dc.log)
	dc.serial.Execute(cmdMedia.StopVideo())
	time.Sleep(200 * time.Millisecond)
	dc.serial.Flush() // clear StopVideo response before issuing play

	cmdStorage := command.NewStorage(dc.log)
	_, err := dc.serial.Execute(cmdStorage.PlayVideo(path, true))
	if err != nil {
		return fmt.Errorf("play video failed: %w", err)
	}
	dc.log.Infof("API: playing video %s", path)
	return nil
}

func (dc *DaemonController) StopVideo() error {
	if err := dc.requireEditor(); err != nil {
		return err
	}
	dc.serial.Flush()
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
