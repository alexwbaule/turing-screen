package initializer

import (
	"fmt"
	"image"
	"image/draw"
	"os"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	themeEntity "github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/video"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
)

// initVideo handles the video mode: check/upload file, start playback, init overlay.
func (i *Initializer) initVideo(videoDisplay *themeEntity.VideoPlay, background device.ImageBackground) error {
	i.log.Info("init: video mode")

	// Check if video exists on device, upload if needed
	if err := i.ensureVideoOnDevice(videoDisplay); err != nil {
		return err
	}

	// Clear display before starting video
	if _, err := i.sender.Execute(i.cmdMedia.StopVideo()); err != nil {
		return fmt.Errorf("STOP_VIDEO (pre-clear) failed: %w", err)
	}
	i.clearDisplay()

	// Start video playback
	if _, err := i.sender.Execute(i.cmdMedia.StartVideo(videoDisplay.Path)); err != nil {
		return fmt.Errorf("START_VIDEO failed: %w", err)
	}
	i.log.Info("init: video started")

	// Initialize overlay
	if err := i.initOverlay(videoDisplay, background); err != nil {
		return err
	}

	i.log.Info("init: video mode complete")
	return nil
}

// ensureVideoOnDevice checks if the video file exists on the device.
// If not, uploads it from the local filesystem.
func (i *Initializer) ensureVideoOnDevice(videoDisplay *themeEntity.VideoPlay) error {
	i.log.Info("init: checking video file on device...")

	cmd := i.cmdStorage.GetFileInfo(videoDisplay.Path)
	resp, err := i.sender.Execute(cmd)
	if err != nil {
		return fmt.Errorf("GET_FILE_INFO failed: %w", err)
	}

	videoSize, err := utils.ParseFileSize(resp)
	if err != nil {
		return fmt.Errorf("failed to parse file size: %w", err)
	}
	i.log.Infof("init: device video file size: %d", videoSize)

	if videoSize == 0 {
		return i.uploadVideo(videoDisplay)
	}

	i.log.Info("init: video already on device")
	return nil
}

// uploadVideo reads the local video file and uploads it to the device.
func (i *Initializer) uploadVideo(videoDisplay *themeEntity.VideoPlay) error {
	i.log.Info("init: uploading video...")

	fileInfo, err := os.Stat(videoDisplay.Path)
	if err != nil {
		return fmt.Errorf("local video file not found '%s': %w", videoDisplay.Path, err)
	}
	localSize := fileInfo.Size()
	i.log.Infof("init: uploading %s (%d bytes)", videoDisplay.Path, localSize)

	createCmd := i.cmdStorage.CreateFile(videoDisplay.Path, localSize)
	if _, err := i.sender.Execute(createCmd); err != nil {
		return fmt.Errorf("CREATE_FILE failed: %w", err)
	}

	videoBytes, err := os.ReadFile(videoDisplay.Path)
	if err != nil {
		return fmt.Errorf("failed to read video file: %w", err)
	}

	uploadCmd := i.cmdStorage.UploadFile(videoBytes)
	if _, err := i.sender.Execute(uploadCmd); err != nil {
		return fmt.Errorf("UPLOAD_FILE failed: %w", err)
	}

	i.log.Info("init: upload complete")
	return nil
}

// initOverlay creates the overlay buffer and sends the initial overlay to the device.
func (i *Initializer) initOverlay(videoDisplay *themeEntity.VideoPlay, background device.ImageBackground) error {
	display := i.cfg.GetDeviceDisplay()
	orientation := i.theme.GetDisplay().Orientation

	var w, h int
	switch orientation {
	case themeEntity.LANDSCAPE, themeEntity.REVERSE_LANDSCAPE:
		w = display.Width
		h = display.Height
	default:
		w = display.Height
		h = display.Width
	}

	// Build overlay background
	scaled := i.buildOverlayBackground(videoDisplay, background.GetImage(), w, h)

	// Build BGRA payload
	bgraPayload := video.BuildInitPayload(scaled, orientation)

	// Create overlay buffer
	overlay := video.NewOverlayBuffer(i.log, display, orientation)
	overlay.SetInitial(scaled)
	i.overlay = overlay

	// Send to device
	initCmd := command.NewInitVideoOverlay(i.log, bgraPayload)
	if _, err := i.sender.Execute(initCmd); err != nil {
		return fmt.Errorf("INIT_VIDEO_OVERLAY failed: %w", err)
	}
	i.log.Info("init: overlay sent")

	// Initial refresh
	refreshCmd := overlay.Refresh()
	if _, err := i.sender.Execute(refreshCmd); err != nil {
		return fmt.Errorf("initial overlay refresh failed: %w", err)
	}

	// Wait and query status
	time.Sleep(1 * time.Second)
	if _, err := i.sender.Execute(i.cmdMedia.QueryStatus()); err != nil {
		i.log.Warnf("init: query status failed (non-fatal): %v", err)
	}

	return nil
}

// buildOverlayBackground composites the video foreground with static content.
func (i *Initializer) buildOverlayBackground(videoDisplay *themeEntity.VideoPlay, staticComposite image.Image, w, h int) *image.NRGBA {
	if videoDisplay.ForegroundImage != nil {
		scaled := device.NewScaledNRGBA(videoDisplay.ForegroundImage, w, h)
		draw.Draw(scaled, image.Rect(0, 0, w, h), staticComposite, image.Point{}, draw.Over)
		return scaled
	}
	return device.NewScaledNRGBA(staticComposite, w, h)
}

// clearDisplay sends a blank white image to clear the display.
func (i *Initializer) clearDisplay() {
	display := i.cfg.GetDeviceDisplay()
	w, h := display.Width, display.Height
	if display.Reverse {
		w, h = h, w
	}
	blankImg := device.NewImageProcess(device.NewBlank(w, h))
	if _, err := i.sender.Execute(i.cmdPayload.SendStaticBitmap(blankImg)); err != nil {
		i.log.Warnf("clear display failed (non-fatal): %v", err)
	}
}
