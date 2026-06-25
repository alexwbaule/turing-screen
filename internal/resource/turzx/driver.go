package turzx

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/cipher"
	"crypto/des"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"sync"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/disintegration/imaging"
	"github.com/google/gousb"
)

const VendorID gousb.ID = 0x1cbe

// ProductID maps USB product ID to native (width, height) in portrait orientation.
// All frames are sent in portrait orientation; SendFrame rotates per theme orientation.
// Source: https://github.com/mathoudebine/turing-smart-screen-python
var ProductID = map[gousb.ID][2]int{
	0x0028: {480, 480},
	0x0046: {320, 960},
	0x0050: {720, 1280}, // Turing 5.2" — native portrait 720×1280
	0x0080: {800, 1280},
	0x0088: {480, 1920},
	0x0092: {462, 1920},
	0x0123: {720, 1920},
}

const (
	cmdSync       = 10
	cmdRestartUSB = 11
	cmdBrightness = 14
	cmdFrameRate  = 15
	cmdUploadPNG  = 102
	cmdStopStream = 123

	// H.264 chunk streaming (send_video protocol)
	cmdVideoMode        = 111
	cmdVideoModeAck     = 112
	cmdVideoInit        = 13
	cmdVideoOverlay     = 41
	cmdGetH264ChunkSize = 17
	cmdPlayH264Chunk    = 121
	cmdGetStreamStatus  = 122

	defaultH264ChunkSize = 202752
)

// Driver handles USB bulk communication with TURZX LCD devices.
// Protocol: DES-CBC encrypted 512-byte command packets + raw PNG/JPEG payload.
type Driver struct {
	usbCtx *gousb.Context
	dev    *gousb.Device
	cfg    *gousb.Config
	intf   *gousb.Interface
	epOut  *gousb.OutEndpoint
	epIn   *gousb.InEndpoint

	PID    gousb.ID
	Width  int
	Height int

	mu     sync.Mutex    // serializes all USB writes (video stream + frame sends)
	videoWg sync.WaitGroup // tracks the H264 streaming goroutine; waited in Close()
	log    *logger.Logger

	// Per-frame scratch buffers — allocated once, reused every SendFrame call.
	// These are NOT protected by mu: encoding runs before the lock is taken,
	// and all call paths (compositor loop + init/shutdown) are single-threaded.
	rotBuf  *image.NRGBA // pre-allocated rotation output (lazy-sized on first use)
	pngBuf  bytes.Buffer // PNG output bytes
	zlibBuf bytes.Buffer // zlib/IDAT stream before CRC wrapping
	zlibW   *zlib.Writer // reused zlib writer (Reset'd between frames)
	pngRow  []byte       // [0]=0 (filter=None) + one RGBA row of pixels
}

// NewDriver finds and opens the first TURZX display connected via USB.
// Returns an error if no device is found or cannot be opened.
func NewDriver(log *logger.Logger) (*Driver, error) {
	ctx := gousb.NewContext()

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if desc.Vendor != VendorID {
			return false
		}
		_, ok := ProductID[desc.Product]
		return ok
	})
	if err != nil && len(devs) == 0 {
		ctx.Close()
		return nil, fmt.Errorf("USB enumeration: %w", err)
	}
	if len(devs) == 0 {
		ctx.Close()
		return nil, fmt.Errorf("no TURZX device found")
	}

	dev := devs[0]
	for _, d := range devs[1:] {
		d.Close()
	}

	pid := dev.Desc.Product
	dims := ProductID[pid]

	// Detach any kernel driver that may have claimed this interface
	_ = dev.SetAutoDetach(true)

	cfg, err := dev.Config(1)
	if err != nil {
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("USB config(1): %w", err)
	}

	intf, err := cfg.Interface(0, 0)
	if err != nil {
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("USB interface(0,0): %w", err)
	}

	var epOut *gousb.OutEndpoint
	var epIn *gousb.InEndpoint
	for _, desc := range intf.Setting.Endpoints {
		if desc.Direction == gousb.EndpointDirectionOut && epOut == nil {
			if ep, e := intf.OutEndpoint(desc.Number); e == nil {
				epOut = ep
			}
		} else if desc.Direction == gousb.EndpointDirectionIn && epIn == nil {
			if ep, e := intf.InEndpoint(desc.Number); e == nil {
				epIn = ep
			}
		}
	}
	if epOut == nil || epIn == nil {
		intf.Close()
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("could not find USB bulk endpoints")
	}

	log.Infof("TURZX driver ready: PID=0x%04x, native %dx%d", uint16(pid), dims[0], dims[1])

	return &Driver{
		usbCtx: ctx,
		dev:    dev,
		cfg:    cfg,
		intf:   intf,
		epOut:  epOut,
		epIn:   epIn,
		PID:    pid,
		Width:  dims[0],
		Height: dims[1],
		log:    log,
	}, nil
}

// Close releases all USB resources.
func (d *Driver) Close() {
	// Wait for the H264 streaming goroutine to exit before releasing USB resources.
	// Without this, libusb_exit() races with an in-flight libusb_submit() → SIGSEGV.
	d.videoWg.Wait()

	if d.intf != nil {
		d.intf.Close()
	}
	if d.cfg != nil {
		d.cfg.Close()
	}
	if d.dev != nil {
		d.dev.Close()
	}
	if d.usbCtx != nil {
		d.usbCtx.Close()
	}
}

// HardReset sends cmd 11 to the device, causing a firmware-level restart.
// Python reference disabled this by default ("Do not enable for now") so use with care.
// May cause USB re-enumeration — caller should reconnect after a short delay.
func (d *Driver) HardReset() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.log.Warn("TURZX: sending HardReset (cmd 11) — device will restart")
	_, err := d.sendCmd(cmdRestartUSB)
	return err
}

// Reconnect closes all USB handles and reopens them without touching device firmware.
// Useful when the host-side USB session is stale. Does NOT power-cycle the device.
func (d *Driver) Reconnect() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.log.Info("TURZX: USB reconnect — closing handles")
	if d.intf != nil {
		d.intf.Close()
		d.intf = nil
	}
	if d.cfg != nil {
		d.cfg.Close()
		d.cfg = nil
	}
	if d.dev != nil {
		d.dev.Close()
		d.dev = nil
	}
	if d.usbCtx != nil {
		d.usbCtx.Close()
		d.usbCtx = nil
	}

	d.log.Info("TURZX: USB reconnect — reopening")
	ctx := gousb.NewContext()
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if desc.Vendor != VendorID {
			return false
		}
		_, ok := ProductID[desc.Product]
		return ok
	})
	if err != nil && len(devs) == 0 {
		ctx.Close()
		return fmt.Errorf("USB reconnect enumeration: %w", err)
	}
	if len(devs) == 0 {
		ctx.Close()
		return fmt.Errorf("USB reconnect: no TURZX device found")
	}
	dev := devs[0]
	for _, other := range devs[1:] {
		other.Close()
	}
	_ = dev.SetAutoDetach(true)
	cfg, err := dev.Config(1)
	if err != nil {
		dev.Close()
		ctx.Close()
		return fmt.Errorf("USB reconnect config: %w", err)
	}
	intf, err := cfg.Interface(0, 0)
	if err != nil {
		cfg.Close()
		dev.Close()
		ctx.Close()
		return fmt.Errorf("USB reconnect interface: %w", err)
	}
	var epOut *gousb.OutEndpoint
	var epIn *gousb.InEndpoint
	for _, desc := range intf.Setting.Endpoints {
		if desc.Direction == gousb.EndpointDirectionOut && epOut == nil {
			if ep, e := intf.OutEndpoint(desc.Number); e == nil {
				epOut = ep
			}
		} else if desc.Direction == gousb.EndpointDirectionIn && epIn == nil {
			if ep, e := intf.InEndpoint(desc.Number); e == nil {
				epIn = ep
			}
		}
	}
	if epOut == nil || epIn == nil {
		intf.Close()
		cfg.Close()
		dev.Close()
		ctx.Close()
		return fmt.Errorf("USB reconnect: endpoints not found")
	}
	d.usbCtx = ctx
	d.dev = dev
	d.cfg = cfg
	d.intf = intf
	d.epOut = epOut
	d.epIn = epIn
	d.log.Infof("TURZX: USB reconnect OK — PID=0x%04x %dx%d", uint16(d.PID), d.Width, d.Height)
	return nil
}

// buildPacket creates the 500-byte command packet header.
// Layout: [0]=cmdID, [2]=0x1A, [3]=0x6D, [4:8]=ms-since-midnight LE, rest=0.
func buildPacket(cmdID int) []byte {
	pkt := make([]byte, 500)
	pkt[0] = byte(cmdID)
	pkt[2] = 0x1A
	pkt[3] = 0x6D
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	binary.LittleEndian.PutUint32(pkt[4:8], uint32(now.Sub(midnight).Milliseconds()))
	return pkt
}

// encryptPacket DES-CBC encrypts a 500-byte packet (key=IV=b"slv3tuzx") → 512 bytes.
// Final bytes: [510]=0xA1, [511]=0x1A (protocol markers).
func encryptPacket(pkt []byte) ([]byte, error) {
	key := []byte("slv3tuzx")
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	// Pad to 8-byte boundary: 500 → 504
	padded := make([]byte, (len(pkt)+7)/8*8)
	copy(padded, pkt)
	cipher.NewCBCEncrypter(block, key).CryptBlocks(padded, padded)

	result := make([]byte, 512)
	copy(result, padded)
	result[510] = 0xA1
	result[511] = 0x1A
	return result, nil
}

// decryptPacket DES-CBC decrypts a response from the device (same key/IV as encryptPacket).
// Returns the decrypted bytes, or nil on error.
func decryptPacket(data []byte) []byte {
	key := []byte("slv3tuzx")
	block, err := des.NewCipher(key)
	if err != nil {
		return nil
	}
	// Must be a multiple of 8
	size := (len(data) / 8) * 8
	if size == 0 {
		return nil
	}
	out := make([]byte, size)
	cipher.NewCBCDecrypter(block, key).CryptBlocks(out, data[:size])
	return out
}

// flush drains any leftover data from the IN endpoint (up to 5 attempts × 100ms).
func (d *Driver) flush() {
	buf := make([]byte, 512)
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		_, err := d.epIn.ReadContext(ctx, buf)
		cancel()
		if err != nil {
			break
		}
	}
}

// sendCmd sends an encrypted command packet and reads the 512-byte response.
// Caller must hold d.mu.
func (d *Driver) sendCmd(cmdID int) ([]byte, error) {
	enc, err := encryptPacket(buildPacket(cmdID))
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	if _, err := d.epOut.Write(enc); err != nil {
		return nil, fmt.Errorf("USB write: %w", err)
	}
	resp := make([]byte, 512)
	n, err := d.epIn.Read(resp)
	if err != nil {
		d.log.Warnf("TURZX read response (non-fatal): %v", err)
		return nil, nil
	}
	d.flush()
	return resp[:n], nil
}

// sendCmdByte sends an encrypted command with a single byte parameter at pkt[8].
// Caller must hold d.mu.
func (d *Driver) sendCmdByte(cmdID int, val byte) error {
	pkt := buildPacket(cmdID)
	pkt[8] = val
	enc, err := encryptPacket(pkt)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	if _, err := d.epOut.Write(enc); err != nil {
		return fmt.Errorf("USB write: %w", err)
	}
	resp := make([]byte, 512)
	d.epIn.Read(resp) //nolint:errcheck
	d.flush()
	return nil
}

// sendImageData sends an encrypted command header followed by raw image bytes.
// Image size is big-endian in pkt[8:12].
// Caller must hold d.mu.
func (d *Driver) sendImageData(cmdID int, data []byte) error {
	pkt := buildPacket(cmdID)
	sz := len(data)
	pkt[8] = byte(sz >> 24)
	pkt[9] = byte(sz >> 16)
	pkt[10] = byte(sz >> 8)
	pkt[11] = byte(sz)
	enc, err := encryptPacket(pkt)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	payload := make([]byte, len(enc)+len(data))
	copy(payload, enc)
	copy(payload[len(enc):], data)
	if _, err := d.epOut.Write(payload); err != nil {
		return fmt.Errorf("USB write: %w", err)
	}
	resp := make([]byte, 512)
	d.epIn.Read(resp) //nolint:errcheck
	d.flush()
	return nil
}

// sendH264Chunk sends one H.264 data chunk via CMD_PLAY_H264_CHUNK (121).
// isLast=true sets pkt[12]=1, signalling end of one full pass to the device.
// Returns the response bytes, or nil on error.
// Caller must hold d.mu.
func (d *Driver) sendH264Chunk(data []byte, isLast bool) []byte {
	pkt := buildPacket(cmdPlayH264Chunk)
	sz := len(data)
	pkt[8] = byte(sz >> 24)
	pkt[9] = byte(sz >> 16)
	pkt[10] = byte(sz >> 8)
	pkt[11] = byte(sz)
	if isLast {
		pkt[12] = 1
	}
	enc, err := encryptPacket(pkt)
	if err != nil {
		d.log.Warnf("TURZX H264 encrypt: %v", err)
		return nil
	}
	payload := make([]byte, len(enc)+len(data))
	copy(payload, enc)
	copy(payload[len(enc):], data)
	if _, err := d.epOut.Write(payload); err != nil {
		d.log.Warnf("TURZX H264 write: %v", err)
		return nil
	}
	resp := make([]byte, 512)
	n, err := d.epIn.Read(resp)
	if err != nil {
		return nil
	}
	d.flush()
	return resp[:n]
}

// Initialize sends SYNC, sets brightness, and sets frame rate to 25 fps.
func (d *Driver) Initialize(brightness int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.log.Info("TURZX: SYNC")
	if _, err := d.sendCmd(cmdSync); err != nil {
		return fmt.Errorf("SYNC: %w", err)
	}
	bval := byte(brightness * 102 / 100)
	d.log.Infof("TURZX: BRIGHTNESS=%d", bval)
	if err := d.sendCmdByte(cmdBrightness, bval); err != nil {
		return fmt.Errorf("BRIGHTNESS: %w", err)
	}
	d.log.Info("TURZX: FRAME_RATE=25")
	if err := d.sendCmdByte(cmdFrameRate, 25); err != nil {
		return fmt.Errorf("FRAME_RATE: %w", err)
	}
	return nil
}

// SendFrame rotates img to the device's native portrait orientation and uploads it.
//
// Device native is PORTRAIT 720×1280 (PID 0x0050). Rotation matches Python reference:
//
//	REVERSE_PORTRAIT  → no rotation (native)
//	LANDSCAPE         → Rotate270 (1280×720 → 720×1280)
//	REVERSE_LANDSCAPE → Rotate90  (1280×720 → 720×1280, flipped)
//	PORTRAIT          → Rotate180
func (d *Driver) SendFrame(img image.Image, orientation theme.Orientation) error {
	// Produce a *image.NRGBA we own so encodeFramePNG can set Pix[3]=254 in-place.
	// Fast path (compositor always sends *image.NRGBA): custom rotate into d.rotBuf —
	// pre-allocated, no heap alloc after the first frame.
	// Slow path (other image types): fall back to imaging.Rotate* (allocates).
	var out *image.NRGBA
	nrgbaSrc, isNRGBA := img.(*image.NRGBA)
	switch orientation {
	case theme.LANDSCAPE:
		if isNRGBA {
			out = d.rotate270(nrgbaSrc)
		} else {
			out = imaging.Rotate270(img)
		}
	case theme.REVERSE_LANDSCAPE:
		if isNRGBA {
			out = d.rotate90(nrgbaSrc)
		} else {
			out = imaging.Rotate90(img)
		}
	case theme.PORTRAIT:
		if isNRGBA {
			out = d.rotate180(nrgbaSrc)
		} else {
			out = imaging.Rotate180(img)
		}
	default: // REVERSE_PORTRAIT — native portrait; clone so we don't mutate compositor's buffer
		out = imaging.Clone(img)
	}
	if out.Bounds().Dx() != d.Width || out.Bounds().Dy() != d.Height {
		fitted := imaging.Fit(out, d.Width, d.Height, imaging.Lanczos)
		canvas := imaging.New(d.Width, d.Height, color.Black)
		ox := (d.Width - fitted.Bounds().Dx()) / 2
		oy := (d.Height - fitted.Bounds().Dy()) / 2
		out = imaging.Paste(canvas, fitted, image.Pt(ox, oy))
	}

	pngBytes, err := d.encodeFramePNG(out)
	if err != nil {
		return fmt.Errorf("PNG encode: %w", err)
	}
	d.log.Debugf("TURZX: sending frame %d B PNG", len(pngBytes))
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sendImageData(cmdUploadPNG, pngBytes)
}

// SetBrightness changes display brightness (0–100 %) after initialization.
func (d *Driver) SetBrightness(value int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sendCmdByte(cmdBrightness, byte(value*102/100))
}

// ClearScreen sends a full black PNG frame to visually blank the display.
// Used as part of ScreenOff (TurnOff). Generates a black image at the device's
// native resolution so it always matches regardless of device variant.
func (d *Driver) ClearScreen() error {
	black := imaging.New(d.Width, d.Height, color.Black)
	pngBytes, err := d.encodeFramePNG(black)
	if err != nil {
		return fmt.Errorf("ClearScreen encode: %w", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sendImageData(cmdUploadPNG, pngBytes)
}

// StopVideoStream sends CMD_STOP_STREAM (cmd 123) to halt video playback from storage.
func (d *Driver) StopVideoStream() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.sendCmd(cmdStopStream)
	return err
}

// StartVideoStream streams h264Data to the device following the Python send_video protocol:
// setup sequence (111→112→13→brightness→41→clear_image→frame_rate→chunk_size_negotiation),
// then a continuous loop sending H264 chunks via cmd 121 until ctx is cancelled.
// PNG overlay frames from SendFrame run between chunks, serialised by d.mu.
func (d *Driver) StartVideoStream(ctx context.Context, h264Data []byte) {
	if len(h264Data) == 0 {
		return
	}

	// Setup sequence — mirrors Python send_video exactly:
	// 111 (VIDEO_MODE), 112 (VIDEO_MODE_ACK), 13 (VIDEO_INIT),
	// 14 (BRIGHTNESS=32), 41 (VIDEO_OVERLAY), 102 (CLEAR_IMAGE), 15 (FRAME_RATE=25),
	// then negotiate chunk size via cmd 17.
	d.mu.Lock()
	d.sendCmd(cmdVideoMode)    //nolint:errcheck
	d.sendCmd(cmdVideoModeAck) //nolint:errcheck
	d.sendCmd(cmdVideoInit)    //nolint:errcheck
	d.sendCmdByte(cmdBrightness, 32) //nolint:errcheck
	d.sendCmd(cmdVideoOverlay) //nolint:errcheck
	d.sendImageData(cmdUploadPNG, d.blackPNG()) //nolint:errcheck // clear_image
	d.sendCmdByte(cmdFrameRate, 25) //nolint:errcheck
	resp, _ := d.sendCmd(cmdGetH264ChunkSize)
	d.mu.Unlock()

	chunkSize := defaultH264ChunkSize
	if resp != nil && len(resp) >= 12 {
		if n := int(binary.BigEndian.Uint32(resp[8:12])); n > 0 && n <= 1024*1024 {
			chunkSize = n
		}
	}
	d.log.Infof("TURZX: H264 streaming, chunk=%d B, total=%d B", chunkSize, len(h264Data))

	d.videoWg.Add(1)
	go func() {
		// videoWg.Done is the outermost defer so it runs AFTER cmdStopStream completes,
		// guaranteeing Close() only proceeds once all USB activity has finished.
		defer d.videoWg.Done()
		defer func() {
			d.mu.Lock()
			d.sendCmd(cmdStopStream) //nolint:errcheck
			d.mu.Unlock()
		}()

		for {
			if ctx.Err() != nil {
				return
			}
			offset := 0
			for offset < len(h264Data) {
				if ctx.Err() != nil {
					return
				}
				end := offset + chunkSize
				if end > len(h264Data) {
					end = len(h264Data)
				}
				isLast := end >= len(h264Data)

				d.mu.Lock()
				resp := d.sendH264Chunk(h264Data[offset:end], isLast)
				var queueDepth byte
				if resp != nil {
					if st, _ := d.sendCmd(cmdGetStreamStatus); st != nil && len(st) > 8 {
						queueDepth = st[8]
					}
				}
				d.mu.Unlock()

				if resp == nil || queueDepth > 3 {
					select {
					case <-time.After(50 * time.Millisecond):
					case <-ctx.Done():
						return
					}
				}
				offset = end
			}
		}
	}()
}

// blackPNG returns a 1×1 black RGBA PNG — used as clear_image in the video setup sequence.
func (d *Driver) blackPNG() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, d.Width, d.Height))
	b, _ := d.encodeFramePNG(img)
	return b
}

// pngSig is the mandatory 8-byte PNG file signature.
var pngSig = []byte{137, 80, 78, 71, 13, 10, 26, 10}

// encodeFramePNG writes nrgba as a valid RGBA PNG (color type 6) using filter=None
// for all rows. Go's stdlib png encoder tries Paeth/Sub/Average filters per row to
// improve compression (~43% of former CPU cost); we skip that entirely. filter=None
// stores raw RGBA bytes — valid PNG, slightly larger but much cheaper to produce.
//
// Alpha[3] is set to 254 in-place so Opaque() returns false, forcing RGBA color type.
// (Go's encoder downgrades fully-opaque images to RGB type 2; device firmware requires type 6.)
// Callers must own the buffer (rotation always produces a new allocation; REVERSE_PORTRAIT clones).
//
// All scratch buffers (pngBuf, zlibBuf, pngRow, zlibW) are struct fields — allocated once,
// reused every call. After the first frame there are zero heap allocations in this function.
func (d *Driver) encodeFramePNG(nrgba *image.NRGBA) ([]byte, error) {
	b := nrgba.Bounds()
	w, h := b.Dx(), b.Dy()

	if w > 0 && h > 0 {
		nrgba.Pix[3] = 254 // prevent Opaque()=true → keep RGBA color type 6
	}

	// Grow row scratch to hold 1 filter byte + one RGBA row; byte [0] stays 0 (None).
	rowBytes := 1 + w*4
	if len(d.pngRow) < rowBytes {
		d.pngRow = make([]byte, rowBytes) // d.pngRow[0] = 0 by default
	}

	// Reset/init reusable buffers.
	d.pngBuf.Reset()
	d.zlibBuf.Reset()
	if d.zlibW == nil {
		d.zlibW, _ = zlib.NewWriterLevel(&d.zlibBuf, zlib.BestSpeed)
	} else {
		d.zlibW.Reset(&d.zlibBuf)
	}

	d.pngBuf.Write(pngSig)

	// IHDR chunk (13 bytes): width, height, bit-depth=8, color-type=6 (RGBA).
	var ihdr [13]byte
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(h))
	ihdr[8] = 8 // bit depth
	ihdr[9] = 6 // color type: RGBA (truecolor + alpha)
	// [10]=0 compression method, [11]=0 filter method, [12]=0 interlace: none
	d.writePNGChunk("IHDR", ihdr[:])

	// IDAT chunk: zlib-compressed rows, each prefixed with filter byte 0 (None).
	for y := b.Min.Y; y < b.Max.Y; y++ {
		rowStart := nrgba.PixOffset(b.Min.X, y)
		copy(d.pngRow[1:rowBytes], nrgba.Pix[rowStart:rowStart+w*4])
		d.zlibW.Write(d.pngRow[:rowBytes]) //nolint:errcheck
	}
	d.zlibW.Close() //nolint:errcheck
	d.writePNGChunk("IDAT", d.zlibBuf.Bytes())

	// IEND chunk (empty body).
	d.writePNGChunk("IEND", nil)

	return d.pngBuf.Bytes(), nil
}

// writePNGChunk appends a PNG chunk to d.pngBuf: 4-byte length, 4-byte name, data, 4-byte CRC32.
func (d *Driver) writePNGChunk(name string, data []byte) {
	var tmp [4]byte
	binary.BigEndian.PutUint32(tmp[:], uint32(len(data)))
	d.pngBuf.Write(tmp[:])
	d.pngBuf.WriteString(name)
	d.pngBuf.Write(data)
	h := crc32.New(crc32.IEEETable)
	h.Write([]byte(name))
	h.Write(data)
	binary.BigEndian.PutUint32(tmp[:], h.Sum32())
	d.pngBuf.Write(tmp[:])
}

// rotate270 matches imaging.Rotate270: 270° CCW = 90° CW.
// src(x, y) → dst(sh-1-y, x). Output: src.Dy() × src.Dx().
// d.rotBuf is allocated once on first call and reused; no heap alloc after that.
func (d *Driver) rotate270(src *image.NRGBA) *image.NRGBA {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	dw, dh := sh, sw // output: H×W

	if d.rotBuf == nil || d.rotBuf.Bounds().Dx() != dw || d.rotBuf.Bounds().Dy() != dh {
		d.rotBuf = image.NewNRGBA(image.Rect(0, 0, dw, dh))
	}
	dst := d.rotBuf
	db := dst.Bounds()

	// dst_x = sh-1-y (fixed per src row), dst_y = x
	// Src rows read sequentially (cache-friendly); dst column writes are strided.
	for y := 0; y < sh; y++ {
		srcOff := src.PixOffset(sb.Min.X, sb.Min.Y+y)
		dstCol := sh - 1 - y
		for x := 0; x < sw; x++ {
			dstOff := dst.PixOffset(db.Min.X+dstCol, db.Min.Y+x)
			dst.Pix[dstOff] = src.Pix[srcOff]
			dst.Pix[dstOff+1] = src.Pix[srcOff+1]
			dst.Pix[dstOff+2] = src.Pix[srcOff+2]
			dst.Pix[dstOff+3] = src.Pix[srcOff+3]
			srcOff += 4
		}
	}
	return dst
}

// rotate90 matches imaging.Rotate90: 90° CCW = 270° CW.
// src(x, y) → dst(y, sw-1-x). Output: src.Dy() × src.Dx().
func (d *Driver) rotate90(src *image.NRGBA) *image.NRGBA {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	dw, dh := sh, sw

	if d.rotBuf == nil || d.rotBuf.Bounds().Dx() != dw || d.rotBuf.Bounds().Dy() != dh {
		d.rotBuf = image.NewNRGBA(image.Rect(0, 0, dw, dh))
	}
	dst := d.rotBuf
	db := dst.Bounds()

	// dst_x = y (fixed per src row), dst_y = sw-1-x
	for y := 0; y < sh; y++ {
		srcOff := src.PixOffset(sb.Min.X, sb.Min.Y+y)
		dstCol := y
		for x := 0; x < sw; x++ {
			dstOff := dst.PixOffset(db.Min.X+dstCol, db.Min.Y+(sw-1-x))
			dst.Pix[dstOff] = src.Pix[srcOff]
			dst.Pix[dstOff+1] = src.Pix[srcOff+1]
			dst.Pix[dstOff+2] = src.Pix[srcOff+2]
			dst.Pix[dstOff+3] = src.Pix[srcOff+3]
			srcOff += 4
		}
	}
	return dst
}

// rotate180 rotates src 180° into d.rotBuf (same dimensions, reversed pixel order).
func (d *Driver) rotate180(src *image.NRGBA) *image.NRGBA {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()

	if d.rotBuf == nil || d.rotBuf.Bounds().Dx() != sw || d.rotBuf.Bounds().Dy() != sh {
		d.rotBuf = image.NewNRGBA(image.Rect(0, 0, sw, sh))
	}
	dst := d.rotBuf
	db := dst.Bounds()

	// Rotate180: src(x, y) → dst(sw-1-x, sh-1-y)
	total := sw * sh
	for i := 0; i < total; i++ {
		x := i % sw
		y := i / sw
		srcOff := src.PixOffset(sb.Min.X+x, sb.Min.Y+y)
		dstOff := dst.PixOffset(db.Min.X+(sw-1-x), db.Min.Y+(sh-1-y))
		dst.Pix[dstOff] = src.Pix[srcOff]
		dst.Pix[dstOff+1] = src.Pix[srcOff+1]
		dst.Pix[dstOff+2] = src.Pix[srcOff+2]
		dst.Pix[dstOff+3] = src.Pix[srcOff+3]
	}
	return dst
}
