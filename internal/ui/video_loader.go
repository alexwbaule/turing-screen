package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

const (
	videoWidth  = 800
	videoHeight = 480
	videoFPS    = 24
)

// VideoPlayer streams video from ffmpeg as raw RGBA and renders via canvas.Raster.
type VideoPlayer struct {
	raster       *canvas.Raster
	currentFrame image.Image
	mu           sync.Mutex
	stopCh       chan struct{}
	running      bool
	videoPath    string
}

// NewVideoPlayer creates a player with a canvas.Raster bound to render the current frame.
func NewVideoPlayer() *VideoPlayer {
	vp := &VideoPlayer{
		stopCh: make(chan struct{}),
	}
	vp.raster = canvas.NewRaster(vp.renderFrame)
	return vp
}

// CanvasObject returns the fyne.CanvasObject for this player (to add to layouts).
func (vp *VideoPlayer) CanvasObject() fyne.CanvasObject {
	return vp.raster
}

// renderFrame is called by Fyne on the render thread — always safe.
func (vp *VideoPlayer) renderFrame(w, h int) image.Image {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	if vp.currentFrame == nil {
		return image.NewUniform(color.Transparent)
	}
	return vp.currentFrame
}

// LoadVideo starts streaming the video via ffmpeg in a loop.
func (vp *VideoPlayer) LoadVideo(videoPath string) error {
	vp.Stop()
	vp.videoPath = videoPath

	// Show first frame immediately
	frame, err := ExtractVideoFrame(videoPath)
	if err != nil {
		return err
	}
	vp.mu.Lock()
	vp.currentFrame = frame
	vp.mu.Unlock()
	canvas.Refresh(vp.raster)

	go vp.streamLoop()
	return nil
}

// Stop halts playback.
func (vp *VideoPlayer) Stop() {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	if vp.running {
		close(vp.stopCh)
		vp.stopCh = make(chan struct{})
		vp.running = false
	}
}

func (vp *VideoPlayer) streamLoop() {
	vp.mu.Lock()
	vp.running = true
	stopCh := vp.stopCh
	vp.mu.Unlock()

	frameSize := videoWidth * videoHeight * 4 // RGBA

	cmd := exec.Command("ffmpeg",
		"-stream_loop", "-1", // loop forever
		"-i", vp.videoPath,
		"-vf", fmt.Sprintf("fps=%d,scale=%d:%d", videoFPS, videoWidth, videoHeight),
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[VideoPlayer] ERRO pipe: %v", err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[VideoPlayer] ERRO start: %v", err)
		return
	}
	defer cmd.Process.Kill()

	buf := make([]byte, frameSize)
	frameDuration := time.Second / time.Duration(videoFPS)
	log.Printf("[VideoPlayer] Streaming %s @ %dfps %dx%d (frame interval: %v)", vp.videoPath, videoFPS, videoWidth, videoHeight, frameDuration)

	for {
		frameStart := time.Now()

		// Check stop
		select {
		case <-stopCh:
			return
		default:
		}

		// Read exactly one frame of RGBA data
		_, err := io.ReadFull(stdout, buf)
		if err != nil {
			log.Printf("[VideoPlayer] Stream ended: %v", err)
			return
		}

		// Create image from raw RGBA
		img := image.NewRGBA(image.Rect(0, 0, videoWidth, videoHeight))
		copy(img.Pix, buf)

		vp.mu.Lock()
		vp.currentFrame = img
		vp.mu.Unlock()

		// Thread-safe: schedule Refresh on Fyne main thread
		fyne.Do(func() {
			canvas.Refresh(vp.raster)
		})

		// Throttle to target FPS — sleep for remaining frame time
		elapsed := time.Since(frameStart)
		if elapsed < frameDuration {
			select {
			case <-stopCh:
				return
			case <-time.After(frameDuration - elapsed):
			}
		}
	}
}

// ExtractVideoFrame extracts the first frame of a video.
func ExtractVideoFrame(videoPath string) (image.Image, error) {
	cmd := exec.Command("ffmpeg",
		"-ss", "0",
		"-i", videoPath,
		"-vframes", "1",
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-vf", fmt.Sprintf("scale=%d:%d", videoWidth, videoHeight),
		"-",
	)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg falhou: %w", err)
	}

	frameSize := videoWidth * videoHeight * 4
	if out.Len() < frameSize {
		return nil, fmt.Errorf("frame muito pequeno: %d bytes", out.Len())
	}

	img := image.NewRGBA(image.Rect(0, 0, videoWidth, videoHeight))
	copy(img.Pix, out.Bytes()[:frameSize])
	return img, nil
}

// CheckFFmpegAvailable checks if ffmpeg and ffprobe are installed.
func CheckFFmpegAvailable() error {
	if err := exec.Command("ffmpeg", "-version").Run(); err != nil {
		return fmt.Errorf("ffmpeg não encontrado. Instale com: sudo pacman -S ffmpeg (ou equivalente)")
	}
	if err := exec.Command("ffprobe", "-version").Run(); err != nil {
		return fmt.Errorf("ffprobe não encontrado. Instale com: sudo pacman -S ffmpeg (ou equivalente)")
	}
	return nil
}
