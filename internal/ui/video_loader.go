package ui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

const videoFPS = 24

// VideoPlayer streams video from ffmpeg as raw RGBA and renders via canvas.Image.
// Unlike canvas.Raster, canvas.Image only redraws when explicitly Refresh()ed,
// so Fyne's render loop is not constantly triggered between frames.
type VideoPlayer struct {
	img          *canvas.Image
	currentFrame image.Image
	mu           sync.Mutex
	stopCh       chan struct{}
	running      bool
	videoPath    string
	frameW       int
	frameH       int
}

// NewVideoPlayer creates a player backed by a canvas.Image.
func NewVideoPlayer() *VideoPlayer {
	vp := &VideoPlayer{
		stopCh: make(chan struct{}),
		frameW: 1280,
		frameH: 720,
	}
	vp.img = canvas.NewImageFromImage(nil)
	vp.img.FillMode = canvas.ImageFillStretch
	return vp
}

// SetSize updates the output frame dimensions for subsequent video loads.
func (vp *VideoPlayer) SetSize(w, h int) {
	if w > 0 && h > 0 {
		vp.frameW = w
		vp.frameH = h
	}
}

// SetOrientation kept for backward compatibility.
func (vp *VideoPlayer) SetOrientation(orientation string) {}

// frameDims returns the output frame dimensions.
func (vp *VideoPlayer) frameDims() (w, h int) {
	return vp.frameW, vp.frameH
}

// buildVF builds the ffmpeg -vf filter string.
func (vp *VideoPlayer) buildVF() string {
	return fmt.Sprintf("fps=%d,scale=%d:%d", videoFPS, vp.frameW, vp.frameH)
}

// CanvasObject returns the fyne.CanvasObject for this player.
func (vp *VideoPlayer) CanvasObject() fyne.CanvasObject {
	return vp.img
}

// LoadVideo starts streaming the video via ffmpeg in a loop.
func (vp *VideoPlayer) LoadVideo(videoPath string) error {
	vp.Stop()
	vp.videoPath = videoPath

	// Show first frame immediately
	frame, err := vp.extractFirstFrame(videoPath)
	if err != nil {
		return err
	}
	vp.mu.Lock()
	vp.currentFrame = frame
	vp.img.Image = frame
	vp.mu.Unlock()
	canvas.Refresh(vp.img)

	go vp.streamLoop()
	return nil
}

// Stop halts playback and clears the image.
func (vp *VideoPlayer) Stop() {
	vp.mu.Lock()
	if vp.running {
		close(vp.stopCh)
		vp.stopCh = make(chan struct{})
		vp.running = false
	}
	vp.img.Image = nil
	vp.mu.Unlock()
	canvas.Refresh(vp.img)
}

func (vp *VideoPlayer) streamLoop() {
	vp.mu.Lock()
	vp.running = true
	stopCh := vp.stopCh
	vp.mu.Unlock()

	// Restart ffmpeg each time the video ends naturally (more reliable than
	// -stream_loop -1, which can freeze some codecs or accumulate memory).
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		stopped := vp.streamOnce(stopCh)
		if stopped {
			return
		}
		// Natural EOF — brief pause then restart
		select {
		case <-stopCh:
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// streamOnce runs one pass of ffmpeg (no loop flag). Returns true if stopped
// intentionally via stopCh, false if the stream ended naturally (EOF/error).
func (vp *VideoPlayer) streamOnce(stopCh chan struct{}) (stopped bool) {
	fw, fh := vp.frameDims()
	frameSize := fw * fh * 4 // RGBA

	cmd := exec.Command("ffmpeg",
		"-i", vp.videoPath,
		"-vf", vp.buildVF(),
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[VideoPlayer] ERRO pipe: %v", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[VideoPlayer] ERRO start: %v", err)
		return false
	}
	defer cmd.Process.Kill()

	buf := make([]byte, frameSize)
	frameDuration := time.Second / time.Duration(videoFPS)
	log.Printf("[VideoPlayer] pass %s @ %dfps %dx%d", vp.videoPath, videoFPS, fw, fh)

	for {
		frameStart := time.Now()

		select {
		case <-stopCh:
			return true
		default:
		}

		_, err := io.ReadFull(stdout, buf)
		if err != nil {
			// Natural end of stream — caller will restart
			return false
		}

		img := image.NewRGBA(image.Rect(0, 0, fw, fh))
		copy(img.Pix, buf)

		vp.mu.Lock()
		vp.currentFrame = img
		vp.img.Image = img
		vp.mu.Unlock()

		canvas.Refresh(vp.img)

		elapsed := time.Since(frameStart)
		if elapsed < frameDuration {
			select {
			case <-stopCh:
				return true
			case <-time.After(frameDuration - elapsed):
			}
		}
	}
}

// extractFirstFrame extracts the first frame of the video scaled to the canvas dimensions.
func (vp *VideoPlayer) extractFirstFrame(videoPath string) (image.Image, error) {
	fw, fh := vp.frameDims()
	vf := fmt.Sprintf("scale=%d:%d", fw, fh)

	cmd := exec.Command("ffmpeg",
		"-ss", "0",
		"-i", videoPath,
		"-vframes", "1",
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-vf", vf,
		"-",
	)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg falhou: %w", err)
	}

	frameSize := fw * fh * 4
	if out.Len() < frameSize {
		return nil, fmt.Errorf("frame muito pequeno: %d bytes", out.Len())
	}

	img := image.NewRGBA(image.Rect(0, 0, fw, fh))
	copy(img.Pix, out.Bytes()[:frameSize])
	return img, nil
}

// ExtractVideoFrame extracts the first frame of a video at the given dimensions.
func ExtractVideoFrame(videoPath string, w, h int) (image.Image, error) {
	cmd := exec.Command("ffmpeg",
		"-ss", "0",
		"-i", videoPath,
		"-vframes", "1",
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-vf", fmt.Sprintf("scale=%d:%d", w, h),
		"-",
	)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg falhou: %w", err)
	}

	frameSize := w * h * 4
	if out.Len() < frameSize {
		return nil, fmt.Errorf("frame muito pequeno: %d bytes", out.Len())
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
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
