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

// sensorData holds the plain data fields for all sensor collectors.
// Separating data from the mutex allows a zero-copy snapshot with a single struct assignment.
type sensorData struct {
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

// SensorValues holds the latest values from all sensor collectors.
type SensorValues struct {
	mu   sync.RWMutex
	data sensorData
}

// Snapshot returns a map of current sensor values, safe to call from any goroutine.
func (v *SensorValues) Snapshot() map[string]interface{} {
	v.mu.RLock()
	d := v.data
	v.mu.RUnlock()
	return map[string]interface{}{
		"cpu_percent":     d.CPUPercent,
		"cpu_temp":        d.CPUTemp,
		"cpu_frequency":   d.CPUFrequency,
		"cpu_fan":         d.CPUFan,
		"cpu_power":       d.CPUPower,
		"cpu_voltage":     d.CPUVoltage,
		"cpu_load1":       d.CPULoad1,
		"cpu_load5":       d.CPULoad5,
		"cpu_load15":      d.CPULoad15,
		"gpu_percent":     d.GPUPercent,
		"gpu_temp":        d.GPUTemp,
		"gpu_memory":      d.GPUMemory,
		"gpu_power":       d.GPUPower,
		"gpu_frequency":   d.GPUFrequency,
		"gpu_voltage":     d.GPUVoltage,
		"gpu_fan":         d.GPUFan,
		"mem_percent":     d.MemPercent,
		"mem_used":        d.MemUsed,
		"mem_free":        d.MemFree,
		"swap_percent":    d.SwapPercent,
		"disk_percent":    d.DiskPercent,
		"disk_free":       d.DiskFree,
		"disk_temp":       d.DiskTemp,
		"net_up_speed":    d.NetUpSpeed,
		"net_down_speed":  d.NetDownSpeed,
		"net_uploaded":    d.NetUploaded,
		"net_downloaded":  d.NetDownloaded,
		"wifi_up_speed":   d.WifiUpSpeed,
		"wifi_down_speed": d.WifiDownSpeed,
		"weather_temp":    d.WeatherTemp,
		"weather_desc":    d.WeatherDesc,
		"weather_wind":    d.WeatherWind,
		"volume":          d.Volume,
	}
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
	// 1. Snapshot all current sensor values in one struct assignment
	c.values.mu.RLock()
	vals := c.values.data
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

	dirtyRects = mergeHorizontalRects(dirtyRects)
	c.log.Debugf("compositor: %d dirty rects to send", len(dirtyRects))

	for _, r := range dirtyRects {
		// Crop the dirty region from the new frame
		crop := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(crop, crop.Bounds(), frame, r.Min, draw.Src)

		// Create UPDATE_BITMAP (static mode) or draw on overlay (video mode → NoOp)
		img := device.NewImageProcess(crop)
		cmd := c.cmdUpdate.SendPayloadFrom(img, r.Min.X, r.Min.Y, "compositor")
		if cmd.GetName() == "NOOP" {
			// Video mode: drawing was a side-effect; no command to enqueue
			continue
		}
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

// findDirtyRects compares two frames and returns pixel-tight bounding boxes of
// changed regions. The frame is divided into 32×32 tiles only as an organisational
// convenience (keeps each result rect bounded to one tile, which helps the
// horizontal merge step). Within each tile a single pass over all pixels
// accumulates the exact min/max coordinates of changed pixels — no separate
// "did anything change?" pre-scan is needed, because a tile with no changes
// simply produces tMinX > tMaxX and is silently skipped.
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

			// Single pass: accumulate the pixel-tight bounding box.
			// Initialise to an inverted range — if nothing changes, tMinX > tMaxX
			// and we produce no rect.
			tMinX := bx + bw
			tMinY := by + bh
			tMaxX := bx - 1
			tMaxY := by - 1

			for py := by; py < by+bh; py++ {
				prevOff := prev.PixOffset(bx, py)
				currOff := curr.PixOffset(bx, py)
				for px := 0; px < bw; px++ {
					po := prevOff + px*4
					co := currOff + px*4
					if prev.Pix[po] != curr.Pix[co] ||
						prev.Pix[po+1] != curr.Pix[co+1] ||
						prev.Pix[po+2] != curr.Pix[co+2] ||
						prev.Pix[po+3] != curr.Pix[co+3] {
						absX := bx + px
						if absX < tMinX {
							tMinX = absX
						}
						if absX > tMaxX {
							tMaxX = absX
						}
						if py < tMinY {
							tMinY = py
						}
						if py > tMaxY {
							tMaxY = py
						}
					}
				}
			}

			if tMinX <= tMaxX {
				dirty = append(dirty, image.Rect(tMinX, tMinY, tMaxX+1, tMaxY+1))
			}
		}
	}

	return dirty
}

// mergeHorizontalRects coalesces adjacent dirty blocks on the same row into
// a single wider rectangle, reducing the number of UPDATE_BITMAP commands.
// Input rects must be ordered row-by-row, left to right (as returned by findDirtyRects).
func mergeHorizontalRects(rects []image.Rectangle) []image.Rectangle {
	if len(rects) == 0 {
		return rects
	}
	merged := make([]image.Rectangle, 0, len(rects))
	merged = append(merged, rects[0])
	for _, r := range rects[1:] {
		last := &merged[len(merged)-1]
		// Same Y span and contiguous in X → extend the last rect
		if r.Min.Y == last.Min.Y && r.Max.Y == last.Max.Y && r.Min.X == last.Max.X {
			last.Max.X = r.Max.X
		} else {
			merged = append(merged, r)
		}
	}
	return merged
}

// imageToNRGBA copies an image to NRGBA.
func imageToNRGBA(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	dst := image.NewNRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}
