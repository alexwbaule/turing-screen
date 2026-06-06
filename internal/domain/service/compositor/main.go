package compositor

import (
	"context"
	"image"
	"image/draw"
	"sync"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
)

// SensorValues holds the latest values from all sensor collectors.
type SensorValues struct {
	mu sync.RWMutex

	CPUPercent   float64
	CPUTemp      float64
	CPUFrequency float64
	CPUFan       float64
	CPUPower     float64
	CPUVoltage   float64
	CPULoad1     float64
	CPULoad5     float64
	CPULoad15    float64

	GPUPercent   float64
	GPUTemp      float64
	GPUMemory    float64
	GPUPower     float64
	GPUFrequency float64
	GPUVoltage   float64
	GPUFan       float64

	MemPercent  float64
	MemUsed     uint64
	MemFree     uint64
	SwapPercent float64

	DiskPercent float64
	DiskFree    float64
	DiskTemp    float64

	NetUpSpeed    float64
	NetDownSpeed  float64
	NetUploaded   uint64
	NetDownloaded uint64
	WifiUpSpeed   float64
	WifiDownSpeed float64

	DateHour time.Time
	DateDay  time.Time

	WeatherTemp float64
	WeatherDesc string
	WeatherWind float64

	Volume float64
}

// Compositor renders all sensors into a single frame and sends diffs to the device.
type Compositor struct {
	log       *logger.Logger
	serial    serial.SerialSender
	builder   *renderer.Builder
	stats     *theme.Stats
	values    *SensorValues
	cmdUpdate *command.UpdatePayload
	interval  time.Duration
	prevFrame *image.NRGBA
	display   *theme.Display
	count     int64
	jobs      chan<- command.Command
}

func New(
	log *logger.Logger,
	ser serial.SerialSender,
	builder *renderer.Builder,
	stats *theme.Stats,
	values *SensorValues,
	cmdUpdate *command.UpdatePayload,
	display *theme.Display,
	interval time.Duration,
) *Compositor {
	return &Compositor{
		log:       log,
		serial:    ser,
		builder:   builder,
		stats:     stats,
		values:    values,
		cmdUpdate: cmdUpdate,
		interval:  interval,
		display:   display,
	}
}

// SetJobs sets the jobs channel where UPDATE_BITMAP commands are sent.
func (c *Compositor) SetJobs(jobs chan<- command.Command) {
	c.jobs = jobs
}

// Run starts the compositor render loop.
func (c *Compositor) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// First frame
	c.renderAndSend()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("compositor stopped")
			return ctx.Err()
		case <-ticker.C:
			c.renderAndSend()
		}
	}
}

func (c *Compositor) renderAndSend() {
	// 1. Read all current values
	c.values.mu.RLock()
	vals := *c.values // snapshot
	c.values.mu.RUnlock()

	// 2. Render everything on top of background
	frame := c.renderFrame(&vals)

	// 3. First frame: store the BACKGROUND as reference (what's on the device)
	if c.prevFrame == nil {
		c.prevFrame = imageToNRGBA(c.builder.GetBackground())
		c.log.Debug("compositor: background stored as reference")
	}

	// 4. Diff with previous frame and send dirty regions
	dirtyRects := c.findDirtyRects(c.prevFrame, frame)
	if len(dirtyRects) == 0 {
		return // nothing changed
	}

	c.log.Debugf("compositor: %d dirty rects to send", len(dirtyRects))

	for _, r := range dirtyRects {
		// Crop the dirty region from the new frame
		crop := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(crop, crop.Bounds(), frame, r.Min, draw.Src)

		// Create UPDATE_BITMAP and push to jobs channel (Worker handles sending)
		img := device.NewImageProcess(crop)
		cmd := c.cmdUpdate.SendPayloadFrom(img, r.Min.X, r.Min.Y, "compositor")
		select {
		case c.jobs <- cmd:
		default:
			// Channel full — DON'T update prevFrame so we retry next tick
			c.log.Warn("compositor: jobs channel full, will retry next tick")
			return
		}
	}

	// 5. Store current frame as previous (ALL jobs were queued successfully)
	c.prevFrame = frame
}

// findDirtyRects compares two frames and returns blocks that changed.
func (c *Compositor) findDirtyRects(prev, curr *image.NRGBA) []image.Rectangle {
	const blockSize = 32
	bounds := prev.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	var dirty []image.Rectangle

	for by := 0; by < h; by += blockSize {
		for bx := 0; bx < w; bx += blockSize {
			bw := blockSize
			bh := blockSize
			if bx+bw > w {
				bw = w - bx
			}
			if by+bh > h {
				bh = h - by
			}

			changed := false
			for py := by; py < by+bh && !changed; py++ {
				prevOff := prev.PixOffset(bx, py)
				currOff := curr.PixOffset(bx, py)
				for px := 0; px < bw*4; px++ {
					if prev.Pix[prevOff+px] != curr.Pix[currOff+px] {
						changed = true
						break
					}
				}
			}

			if changed {
				dirty = append(dirty, image.Rect(bx, by, bx+bw, by+bh))
			}
		}
	}

	return dirty
}

// imageToNRGBA copies an image to NRGBA.
func imageToNRGBA(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	dst := image.NewNRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}
