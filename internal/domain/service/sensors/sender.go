package sensors

import (
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/service/video"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
)

// SensorOutput abstracts how sensor data is sent to the display.
type SensorOutput interface {
	Send(img *device.ImageProcess, x, y int)
}

// StaticSender creates UpdatePayload commands and sends them to the jobs channel.
// This is the behavior for static image mode (no video overlay).
type StaticSender struct {
	jobs      chan<- command.Command
	cmdUpdate *command.UpdatePayload
	log       *logger.Logger
}

func NewStaticSender(
	jobs chan<- command.Command,
	cmdUpdate *command.UpdatePayload,
	log *logger.Logger,
) *StaticSender {
	return &StaticSender{
		jobs:      jobs,
		cmdUpdate: cmdUpdate,
		log:       log,
	}
}

func (s *StaticSender) Send(img *device.ImageProcess, x, y int) {
	cmd := s.cmdUpdate.SendPayload(img, x, y)
	select {
	case s.jobs <- cmd:
	default:
		s.log.Warn("static sensor update dropped: jobs channel full")
	}
}

// VideoSender draws sensor images onto the video overlay buffer.
// The overlay refresh goroutine handles periodic diff computation and sending.
type VideoSender struct {
	overlay *video.OverlayBuffer
	log     *logger.Logger
}

func NewVideoSender(overlay *video.OverlayBuffer, log *logger.Logger) *VideoSender {
	return &VideoSender{
		overlay: overlay,
		log:     log,
	}
}

func (s *VideoSender) Send(img *device.ImageProcess, x, y int) {
	s.overlay.Draw(img.GetImage(), x, y)
}
