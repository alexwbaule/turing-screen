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

	"github.com/alexwbaule/turing-screen/internal/application/config"
	"github.com/alexwbaule/turing-screen/internal/application/hwinfo"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	entityTheme "github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
	"github.com/alexwbaule/turing-screen/internal/resource/turzx"
)

// DaemonController implements the Controller interface for the turing-screen daemon.
type DaemonController struct {
	log        *logger.Logger
	mu         sync.Mutex
	mode       string // "starting", "normal", or "editor"
	themeName  string
	firmware   string
	startTime  time.Time
	brightness int // last brightness value sent to the device

	// Serial and commands (Rev-C path; nil for TURZX)
	serial     serial.SerialSender
	cmdDevice  *command.Device
	cmdBright  *command.Brightness
	cmdPayload *command.Payload

	// TURZX USB driver (non-nil when a TURZX device is connected; nil for Rev-C)
	turzxDrv    *turzx.Driver
	orientation entityTheme.Orientation // display orientation for preview frames

	// Sensor control
	stopFunc    func()
	restartFunc func()
	sensorFunc  func() map[string]interface{}

	// Hardware info (detected once at startup / theme reload)
	hwInfo    *hwinfo.HWInfo
	hwInfoCb  func(HWInfoPayload) // called after hwInfo is set so the server can re-broadcast

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

// SetHWInfo stores detected hardware info and notifies the server so it can
// re-broadcast event.hwinfo to all already-connected clients.
func (dc *DaemonController) SetHWInfo(hw *hwinfo.HWInfo) {
	dc.mu.Lock()
	dc.hwInfo = hw
	cb := dc.hwInfoCb
	dc.mu.Unlock()
	if cb != nil {
		cb(dc.GetHWInfo())
	}
}

// OnHWInfoReady registers a callback that fires every time SetHWInfo is called.
// The server uses this to re-push event.hwinfo to connected clients.
func (dc *DaemonController) OnHWInfoReady(fn func(HWInfoPayload)) {
	dc.mu.Lock()
	dc.hwInfoCb = fn
	dc.mu.Unlock()
}

// GetHWInfo returns a snapshot of the detected hardware info.
func (dc *DaemonController) GetHWInfo() HWInfoPayload {
	dc.mu.Lock()
	hw := dc.hwInfo
	dc.mu.Unlock()
	if hw == nil {
		return HWInfoPayload{}
	}
	return HWInfoPayload{
		CPUModel: hw.CPUModel,
		GPUModel: hw.GPUModel,
		MemTotal: hw.MemTotal,
		Hostname: hw.Hostname,
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

// SetTURZXDriver registers the TURZX USB driver. When set, all device operations
// dispatch to TURZX instead of Rev-C serial.
func (dc *DaemonController) SetTURZXDriver(drv *turzx.Driver) {
	dc.mu.Lock()
	dc.turzxDrv = drv
	dc.mu.Unlock()
}

// SetOrientation stores the current theme's display orientation so that
// PreviewImage can rotate frames correctly for the TURZX device.
func (dc *DaemonController) SetOrientation(o entityTheme.Orientation) {
	dc.mu.Lock()
	dc.orientation = o
	dc.mu.Unlock()
}

func (dc *DaemonController) GetStatus() StatusResponse {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	deviceType := "revc"
	if dc.turzxDrv != nil {
		deviceType = "turzx"
	}
	return StatusResponse{
		Mode:       dc.mode,
		Theme:      dc.themeName,
		Firmware:   dc.firmware,
		Uptime:     time.Since(dc.startTime).Round(time.Second).String(),
		APIVersion: "1.0",
		DeviceType: deviceType,
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
	dc.mu.Lock()
	drv := dc.turzxDrv
	dc.mu.Unlock()

	var err error
	if drv != nil {
		err = drv.SetBrightness(value)
	} else {
		_, err = dc.serial.Execute(dc.cmdBright.SetBrightness(value))
	}
	if err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}
	dc.mu.Lock()
	dc.brightness = value
	dc.mu.Unlock()
	dc.log.Infof("API: brightness set to %d", value)
	return nil
}

// SaveSettings persists display settings to device flash.
// TURZX: uses cmd 125 to write all fields atomically.
// Rev-C: only brightness is supported; other fields are logged as warnings.
func (dc *DaemonController) SaveSettings(s DeviceSettings) error {
	dc.mu.Lock()
	drv := dc.turzxDrv
	dc.mu.Unlock()

	if drv != nil {
		bri := byte(s.Brightness * 102 / 100)
		if err := drv.SaveSettings(bri, byte(s.Startup), 0, byte(s.Rotation), byte(s.Sleep), byte(s.Offline)); err != nil {
			return fmt.Errorf("TURZX SaveSettings: %w", err)
		}
		dc.log.Infof("API: TURZX settings saved (brightness=%d startup=%d rotation=%d sleep=%d offline=%d)",
			s.Brightness, s.Startup, s.Rotation, s.Sleep, s.Offline)
		return nil
	}

	// Rev-C only supports brightness
	if s.Startup != 0 || s.Rotation != 0 || s.Sleep != 0 || s.Offline != 0 {
		dc.log.Warnf("API: Rev-C device does not support startup/rotation/sleep/offline settings — ignored")
	}
	if s.Brightness > 0 {
		return dc.SetBrightness(s.Brightness)
	}
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

// RestartDevice for TURZX acts as "wake screen": restores brightness and restarts
// the compositor so the theme reappears. Python explicitly disables cmd 11 (hardware
// restart) for USB models; the only meaningful action is bringing the display back.
// For Rev-C serial it sends the soft-restart command (0x82).
func (dc *DaemonController) RestartDevice() error {
	dc.mu.Lock()
	drv := dc.turzxDrv
	brightness := dc.brightness
	dc.mu.Unlock()

	if drv != nil {
		dc.log.Infof("API: TURZX wake screen — restoring brightness %d and restarting theme", brightness)
		if err := drv.SetBrightness(brightness); err != nil {
			dc.log.Warnf("API: TURZX wake SetBrightness: %v", err)
		}
		dc.mu.Lock()
		dc.mode = "starting"
		dc.mu.Unlock()
		if dc.restartFunc != nil {
			go dc.restartFunc()
		}
		return nil
	}
	dc.log.Info("API: Rev-C restart device (soft 0x82)")
	cmdStorage := command.NewStorage(dc.log)
	dc.serial.Execute(cmdStorage.RestartDevice()) //nolint:errcheck
	return nil
}

// RebootDevice for TURZX is the same as RestartDevice (no hard-reboot command exists).
// For Rev-C serial it sends the hard-reboot command (0x84).
func (dc *DaemonController) RebootDevice() error {
	dc.mu.Lock()
	drv := dc.turzxDrv
	dc.mu.Unlock()

	if drv != nil {
		return dc.RestartDevice()
	}
	dc.log.Info("API: Rev-C reboot device (hard 0x84)")
	cmdDevice := command.NewDevice(dc.log)
	dc.serial.Execute(cmdDevice.Restart()) //nolint:errcheck
	return nil
}

func (dc *DaemonController) ResetUSB() error {
	dc.mu.Lock()
	drv := dc.turzxDrv
	dc.mu.Unlock()

	if drv != nil {
		dc.log.Info("API: TURZX USB reset — stopping sensors then reconnecting USB")
		if dc.stopFunc != nil {
			dc.stopFunc()
		}
		dc.mu.Lock()
		dc.mode = "editor"
		dc.mu.Unlock()

		if err := drv.Reconnect(); err != nil {
			dc.log.Errorf("API: TURZX USB reconnect failed: %v", err)
			return fmt.Errorf("USB reconnect: %w", err)
		}

		dc.log.Info("API: TURZX USB reconnect OK — restarting sensors")
		dc.mu.Lock()
		dc.mode = "starting"
		dc.mu.Unlock()
		if dc.restartFunc != nil {
			go dc.restartFunc()
		}
		return nil
	}

	dc.log.Info("API: Rev-C USB reset — stopping sensors first")
	if dc.stopFunc != nil {
		dc.stopFunc()
	}
	dc.mu.Lock()
	dc.mode = "editor"
	dc.mu.Unlock()

	if err := dc.serial.ResetDevice(); err != nil {
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

// HardResetDevice sends cmd 11 to the TURZX device, triggering a firmware restart.
// Python reference disabled this by default. Use to force SD card re-detection.
// After the call the USB session will be stale — sensors will auto-restart via restartFunc.
func (dc *DaemonController) HardResetDevice() error {
	dc.mu.Lock()
	drv := dc.turzxDrv
	dc.mu.Unlock()

	if drv != nil {
		dc.log.Warn("API: TURZX HardReset (cmd 11) — device will restart, USB session will drop")
		if dc.stopFunc != nil {
			dc.stopFunc()
		}
		dc.mu.Lock()
		dc.mode = "editor"
		dc.mu.Unlock()

		drv.HardReset() //nolint:errcheck — device restarts, response may never arrive

		// Give device time to restart then reconnect
		time.Sleep(3 * time.Second)
		if err := drv.Reconnect(); err != nil {
			dc.log.Errorf("API: TURZX post-hardreset reconnect failed: %v", err)
			return fmt.Errorf("post-reset reconnect: %w", err)
		}
		dc.log.Info("API: TURZX post-hardreset reconnect OK — restarting sensors")
		dc.mu.Lock()
		dc.mode = "starting"
		dc.mu.Unlock()
		if dc.restartFunc != nil {
			go dc.restartFunc()
		}
		return nil
	}

	dc.log.Info("API: Rev-C hard reset (0x84)")
	cmdDevice := command.NewDevice(dc.log)
	dc.serial.Execute(cmdDevice.Restart()) //nolint:errcheck
	return nil
}

// TurnOff for TURZX: stops sensors (halts compositor + video stream), then blanks
// the screen by sending a black frame and setting brightness to 0. The device firmware
// stays alive so RestartDevice() / wake can bring it back without USB re-enumeration.
// For Rev-C serial it sends the hardware power-off command.
func (dc *DaemonController) TurnOff() error {
	dc.mu.Lock()
	stop := dc.stopFunc
	drv := dc.turzxDrv
	dc.mode = "editor"
	dc.mu.Unlock()

	if stop != nil {
		stop()
	}
	if drv != nil {
		dc.log.Info("API: TURZX screen off — clearing display and setting brightness 0")
		if err := drv.ClearScreen(); err != nil {
			dc.log.Warnf("API: TURZX ClearScreen: %v", err)
		}
		if err := drv.SetBrightness(0); err != nil {
			dc.log.Warnf("API: TURZX SetBrightness(0): %v", err)
		}
		return nil
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
	drv := dc.turzxDrv
	orient := dc.orientation
	dc.mu.Unlock()

	img, err := png.Decode(bytes.NewReader(imgData))
	if err != nil {
		return fmt.Errorf("invalid PNG: %w", err)
	}

	if drv != nil {
		if err := drv.SendFrame(toNRGBA(img), orient); err != nil {
			return fmt.Errorf("failed to send preview: %w", err)
		}
		dc.log.Info("API: TURZX preview image sent")
		return nil
	}

	processedImg := device.NewImageProcess(toNRGBA(img))
	_, err = dc.serial.Execute(dc.cmdPayload.SendStaticBitmap(processedImg))
	if err != nil {
		return fmt.Errorf("failed to send preview: %w", err)
	}
	dc.log.Info("API: Rev-C preview image sent to device")
	return nil
}

func (dc *DaemonController) ApplyTheme(name string) error {
	// Persist the theme choice to conf/config.yaml. The restart below re-reads
	// the theme name from disk, so we must write it before resuming sensors.
	if err := config.WriteThemeName(name); err != nil {
		return fmt.Errorf("failed to persist theme %q: %w", name, err)
	}

	// Stop sensors completely before switching theme (SetModeEditor is synchronous).
	if err := dc.SetModeEditor(); err != nil {
		return err
	}

	dc.mu.Lock()
	dc.themeName = name
	dc.mu.Unlock()

	dc.log.Infof("API: theme set to %s, restarting...", name)

	// Resume — startSensors reloads config from disk, which now has the new theme.
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
	dc.log.Info("API: GetStorageInfo called")

	dc.mu.Lock()
	drv := dc.turzxDrv
	dc.mu.Unlock()

	if drv != nil {
		// TURZX: read-only USB command, driver mutex handles concurrency — no editor mode needed.
		dc.log.Info("API: TURZX GetStorageInfo → calling driver")
		info, err := drv.GetStorageInfo()
		if err != nil {
			dc.log.Errorf("API: TURZX GetStorageInfo error: %v", err)
			return StorageInfo{}, err
		}
		dc.log.Infof("API: TURZX GetStorageInfo result: total=%d KB used=%d KB free=%d KB", info.TotalKB, info.UsedKB, info.FreeKB)
		return StorageInfo{
			Total: int64(info.TotalKB) * 1024,
			Used:  int64(info.UsedKB) * 1024,
			Free:  int64(info.FreeKB) * 1024,
		}, nil
	}

	// Rev-C serial path: requires exclusive access to the serial port.
	if err := dc.requireEditor(); err != nil {
		dc.log.Warnf("API: GetStorageInfo blocked by requireEditor: %v", err)
		return StorageInfo{}, err
	}
	dc.log.Info("API: Rev-C GetStorageInfo → serial")
	dc.serial.Flush()
	cmdStorage := command.NewStorage(dc.log)
	var resp []byte
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			dc.serial.Flush()
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

	dc.mu.Lock()
	drv := dc.turzxDrv
	dc.mu.Unlock()

	if drv != nil {
		// Map the caller-supplied path to the TURZX device path convention.
		// UI may pass "video" or "image"; driver uses /tmp/sdcard/mmcblk0p1/{video,img}/
		devicePath := turzx.StorageVideoPath
		if strings.Contains(strings.ToLower(path), "img") || strings.Contains(strings.ToLower(path), "image") {
			devicePath = turzx.StorageImagePath
		}
		dc.log.Infof("API: TURZX GetStorageFiles path=%q → %q", path, devicePath)
		files, err := drv.ListDir(devicePath)
		if err != nil {
			return nil, fmt.Errorf("TURZX ListDir: %w", err)
		}
		return files, nil
	}

	dc.serial.Flush()
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

	raw := strings.TrimRight(string(bytes.Trim(resp, "\x00")), "/")
	if idx := strings.Index(raw, "file:"); idx >= 0 {
		raw = raw[idx+5:]
	} else if idx := strings.Index(raw, "dir:"); idx >= 0 {
		raw = raw[idx+4:]
	}
	if raw == "" {
		return []string{}, nil
	}
	return strings.Split(raw, "/"), nil
}

func (dc *DaemonController) UploadFile(name string, data []byte) error {
	if err := dc.requireEditor(); err != nil {
		return err
	}

	dc.mu.Lock()
	drv := dc.turzxDrv
	brightness := byte(dc.brightness)
	dc.mu.Unlock()

	if drv != nil {
		remotePath := "/tmp/sdcard/mmcblk0p1/video/" + name
		return drv.UploadFile(remotePath, data)
	}

	dc.serial.Flush()
	cmdStorage := command.NewStorage(dc.log)
	if _, err := dc.serial.Execute(cmdStorage.SetPreUpload(brightness)); err != nil {
		return fmt.Errorf("pre-upload failed: %w", err)
	}
	path := "/root/video/" + name
	if _, err := dc.serial.Execute(cmdStorage.CreateFile(path, int64(len(data)))); err != nil {
		return fmt.Errorf("create file failed: %w", err)
	}
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

	dc.mu.Lock()
	drv := dc.turzxDrv
	dc.mu.Unlock()

	if drv != nil {
		return drv.DeleteFile(path)
	}

	dc.serial.Flush()
	cmdStorage := command.NewStorage(dc.log)
	if _, err := dc.serial.Execute(cmdStorage.DeleteFile(path)); err != nil {
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

	dc.mu.Lock()
	drv := dc.turzxDrv
	dc.mu.Unlock()

	if drv != nil {
		if err := drv.PlayVideoFromStorage(path); err != nil {
			return fmt.Errorf("TURZX play video failed: %w", err)
		}
		dc.log.Infof("API: TURZX playing video %s", path)
		return nil
	}

	dc.serial.Flush()
	cmdMedia := command.NewMedia(dc.log)
	dc.serial.Execute(cmdMedia.StopVideo()) //nolint:errcheck
	time.Sleep(200 * time.Millisecond)
	dc.serial.Flush()
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

	dc.mu.Lock()
	drv := dc.turzxDrv
	dc.mu.Unlock()

	if drv != nil {
		if err := drv.StopVideoStream(); err != nil {
			dc.log.Warnf("API: TURZX StopVideo: %v", err)
		}
		dc.log.Info("API: TURZX video stopped")
		return nil
	}

	dc.serial.Flush()
	cmdMedia := command.NewMedia(dc.log)
	dc.serial.Execute(cmdMedia.StopVideo()) //nolint:errcheck
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
