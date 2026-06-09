package initializer

import (
	"fmt"

	"github.com/alexwbaule/turing-screen/internal/application/config"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/service/video"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
)

// Initializer handles the synchronous device initialization sequence.
// It must be run before the async worker starts, because some steps
// require reading responses to make decisions (e.g. check file exists → upload).
type Initializer struct {
	sender     serial.SerialSender
	log        *logger.Logger
	cfg        *config.Config
	theme      *theme.Theme
	cmdDevice  *command.Device
	cmdMedia   *command.Media
	cmdOption  *command.Option
	cmdStorage *command.Storage
	cmdBright  *command.Brightness
	cmdPayload *command.Payload
	overlay    *video.OverlayBuffer
}

// New creates a new Initializer instance.
func New(
	sender serial.SerialSender,
	log *logger.Logger,
	cfg *config.Config,
	theme *theme.Theme,
	cmdDevice *command.Device,
	cmdMedia *command.Media,
	cmdOption *command.Option,
	cmdStorage *command.Storage,
	cmdBright *command.Brightness,
	cmdPayload *command.Payload,
) *Initializer {
	return &Initializer{
		sender:     sender,
		log:        log,
		cfg:        cfg,
		theme:      theme,
		cmdDevice:  cmdDevice,
		cmdMedia:   cmdMedia,
		cmdOption:  cmdOption,
		cmdStorage: cmdStorage,
		cmdBright:  cmdBright,
		cmdPayload: cmdPayload,
	}
}

// Overlay returns the video overlay buffer created during initialization.
// Returns nil if the device is in static image mode.
func (i *Initializer) Overlay() *video.OverlayBuffer {
	return i.overlay
}

// Run executes the synchronous initialization sequence.
// Detects the mode (static or video) and runs the appropriate flow.
func (i *Initializer) Run(background device.ImageBackground) error {
	i.log.Info("starting device initialization...")

	// Common init: handshake + options + stop media + brightness
	if err := i.commonInit(); err != nil {
		return err
	}

	// Detect mode and run the appropriate flow
	videoDisplay := i.theme.GetVideoPlay()
	if videoDisplay != nil {
		return i.initVideo(videoDisplay, background)
	}
	return i.initStatic(background)
}

// commonInit runs the shared initialization steps for both modes.
func (i *Initializer) commonInit() error {
	// Discard stale bytes left in the OS receive buffer from the previous session.
	// Without this, partial responses from prior commands corrupt the HELLO read.
	if err := i.sender.Flush(); err != nil {
		i.log.Warnf("pre-init serial flush failed (non-fatal): %v", err)
	}

	// 1. HELLO + read(23)
	if _, err := i.sender.Execute(i.cmdDevice.Hello()); err != nil {
		return fmt.Errorf("HELLO failed: %w", err)
	}
	i.log.Info("init: HELLO ok")

	// 2. STOP_VIDEO (no response)
	if _, err := i.sender.Execute(i.cmdMedia.StopVideo()); err != nil {
		return fmt.Errorf("STOP_VIDEO failed: %w", err)
	}

	// 3. STOP_MEDIA + read(1024) — single response for both stops
	if _, err := i.sender.Execute(i.cmdMedia.StopMedia()); err != nil {
		i.log.Warnf("STOP_MEDIA read failed (non-fatal): %v", err)
	}

	// 4. SET_BRIGHTNESS (no response)
	brightness := i.cfg.GetDeviceDisplay().Brightness
	if _, err := i.sender.Execute(i.cmdBright.SetBrightness(brightness)); err != nil {
		return fmt.Errorf("SET_BRIGHTNESS failed: %w", err)
	}

	// 5. OPTIONS with orientation (no response)
	// SetOptions returns a new *Option — assign it back so Execute sends the
	// configured bytes (before this fix, the return value was discarded and
	// Execute sent the zero-value struct = 250 null bytes).
	opt := i.cmdOption.SetOptions(command.Default, command.NoFlip, command.Disabled)
	if _, err := i.sender.Execute(opt); err != nil {
		return fmt.Errorf("OPTIONS failed: %w", err)
	}

	i.log.Info("init: media stopped, brightness set")
	return nil
}
