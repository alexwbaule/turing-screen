package turzx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

// StorageInfo holds the parsed storage response from the device (cmd 100).
// Values are in KB (confirmed from Python reference format_bytes which divides by 1024*1024 for GB).
type StorageInfo struct {
	TotalKB uint32
	UsedKB  uint32
	FreeKB  uint32
}

func (s StorageInfo) String() string {
	return fmt.Sprintf("total=%s used=%s free=%s",
		formatKB(s.TotalKB), formatKB(s.UsedKB), formatKB(s.FreeKB))
}

func formatKB(kb uint32) string {
	if kb >= 1024*1024 {
		return fmt.Sprintf("%.2f GB", float64(kb)/(1024*1024))
	}
	return fmt.Sprintf("%.2f MB", float64(kb)/1024)
}

const (
	cmdOpenFile       = 38
	cmdWriteFileChunk = 39
	cmdDeleteFileUSB  = 40
	cmdPlayVideoUSB   = 98
	cmdGetStorageInfo = 100
	cmdListDir        = 101 // 0x65 — same as Rev-C; path must be /tmp/sdcard/mmcblk0p1/{video,img}/
	cmdPlayVideo2     = 110
	cmdPlayImage      = 113
	cmdSaveSettings   = 125

	// Device storage paths (from turing-smart-screen-python lcd_comm_turing_usb.py)
	StorageVideoPath = "/tmp/sdcard/mmcblk0p1/video/"
	StorageImagePath = "/tmp/sdcard/mmcblk0p1/img/"

	uploadChunkSize = 1024 * 1024 // 1 MB per chunk
)

// buildPathPacket builds a 500-byte command packet with a device path embedded at offset 16.
// Layout: [0]=cmdID, [2]=0x1A, [3]=0x6D, [4:8]=time LE, [8:12]=pathLen BE, [12:16]=0x00, [16:]=path.
func buildPathPacket(cmdID int, path string) []byte {
	pkt := buildPacket(cmdID)
	pb := []byte(path)
	if len(pb) > 484 {
		pb = pb[:484]
	}
	pLen := len(pb)
	pkt[8] = byte(pLen >> 24)
	pkt[9] = byte(pLen >> 16)
	pkt[10] = byte(pLen >> 8)
	pkt[11] = byte(pLen)
	// pkt[12:16] already zero
	copy(pkt[16:], pb)
	return pkt
}

// GetStorageInfo queries device storage capacity (cmd 100).
// Response layout (confirmed from Python reference lcd_comm_turing_usb.py):
//   [0]     = cmdID echo
//   [1]     = status byte (0xc8 = OK)
//   [2:8]   = device internal header (timestamp/counter, varies per call)
//   [8:12]  = TotalBytes  LE uint32
//   [12:16] = UsedBytes   LE uint32
//   [16:20] = ValidBytes  LE uint32 (free space)
// All-zero [8:20] means SD card not detected (hot-swap not supported; insert before power-on).
func (d *Driver) GetStorageInfo() (*StorageInfo, error) {
	d.log.Infof("TURZX: GetStorageInfo → sending cmd %d", cmdGetStorageInfo)
	d.mu.Lock()
	resp, err := d.sendCmd(cmdGetStorageInfo)
	d.mu.Unlock()
	if err != nil {
		d.log.Errorf("TURZX: GetStorageInfo cmd error: %v", err)
		return nil, fmt.Errorf("GetStorageInfo: %w", err)
	}
	d.log.Infof("TURZX: GetStorageInfo raw n=%d header=%x storage=%x",
		len(resp), resp[:min(8, len(resp))], resp[8:min(20, len(resp))])
	if len(resp) < 20 {
		return nil, fmt.Errorf("GetStorageInfo: short response (%d bytes)", len(resp))
	}
	info := &StorageInfo{
		TotalKB: binary.LittleEndian.Uint32(resp[8:12]),
		UsedKB:  binary.LittleEndian.Uint32(resp[12:16]),
		FreeKB:  binary.LittleEndian.Uint32(resp[16:20]),
	}
	if info.TotalKB == 0 {
		d.log.Warn("TURZX: GetStorageInfo total=0 — SD card not detected (insert before power-on)")
	} else {
		d.log.Infof("TURZX: GetStorageInfo parsed: %s", info)
	}
	return info, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ListDir returns filenames in the given device directory (cmd 101).
// Use StorageVideoPath or StorageImagePath as path.
// Response format (same as Rev-C ASCII): "result:dir:file:name1/file:name2/"
// Returns empty list (not an error) when the directory is empty or SD card has no files.
func (d *Driver) ListDir(path string) ([]string, error) {
	pkt := buildPathPacket(cmdListDir, path)
	enc, err := encryptPacket(pkt)
	if err != nil {
		return nil, fmt.Errorf("ListDir encrypt: %w", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, err := d.epOut.Write(enc); err != nil {
		return nil, fmt.Errorf("ListDir write: %w", err)
	}
	// Read up to 1024 bytes — response may be larger than a single 512-byte packet.
	resp := make([]byte, 1024)
	n, err := d.epIn.Read(resp)
	if err != nil {
		d.log.Warnf("TURZX ListDir read (non-fatal): %v", err)
		return nil, nil
	}
	d.flush()
	raw := resp[:n]
	d.log.Infof("TURZX ListDir path=%q raw n=%d hex=%x", path, n, raw[:min(32, n)])

	// Parse ASCII response: skip binary header bytes until "result:dir:" prefix.
	text := string(bytes.TrimRight(raw, "\x00"))
	idx := bytes.Index(raw, []byte("result:dir:"))
	if idx < 0 {
		d.log.Warnf("TURZX ListDir: no 'result:dir:' in response — path=%q raw=%x", path, raw[:min(32, n)])
		return nil, nil
	}
	content := text[idx+len("result:dir:"):]
	content = strings.TrimSuffix(strings.ReplaceAll(content, "file:", ""), "/")
	if content == "" {
		return nil, nil
	}
	var files []string
	for _, f := range strings.Split(content, "/") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

// SaveSettings persists display settings to device flash (cmd 125).
func (d *Driver) SaveSettings(brightness, startup, reserved, rotation, sleep, offline byte) error {
	pkt := buildPacket(cmdSaveSettings)
	pkt[8] = brightness
	pkt[9] = startup
	pkt[10] = reserved
	pkt[11] = rotation
	pkt[12] = sleep
	pkt[13] = offline
	enc, err := encryptPacket(pkt)
	if err != nil {
		return fmt.Errorf("SaveSettings encrypt: %w", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, err := d.epOut.Write(enc); err != nil {
		return fmt.Errorf("SaveSettings write: %w", err)
	}
	buf := make([]byte, 512)
	d.epIn.Read(buf) //nolint:errcheck
	d.flush()
	return nil
}

// openFile sends cmd 38 to create/open a remote file path for writing.
// Caller must hold d.mu.
func (d *Driver) openFile(remotePath string) error {
	pkt := buildPathPacket(cmdOpenFile, remotePath)
	enc, err := encryptPacket(pkt)
	if err != nil {
		return fmt.Errorf("openFile encrypt: %w", err)
	}
	if _, err := d.epOut.Write(enc); err != nil {
		return fmt.Errorf("openFile write: %w", err)
	}
	resp := make([]byte, 512)
	d.epIn.Read(resp) //nolint:errcheck
	d.flush()
	return nil
}

// writeFileChunk sends cmd 39 with a data chunk.
// Layout: [8:12]=capacity BE, [12:16]=chunkLen BE, [16]=isLast, payload=chunk.
// Caller must hold d.mu.
func (d *Driver) writeFileChunk(chunk []byte, isLast bool) error {
	pkt := buildPacket(cmdWriteFileChunk)
	cap := uploadChunkSize
	pkt[8] = byte(cap >> 24)
	pkt[9] = byte(cap >> 16)
	pkt[10] = byte(cap >> 8)
	pkt[11] = byte(cap)
	cl := len(chunk)
	pkt[12] = byte(cl >> 24)
	pkt[13] = byte(cl >> 16)
	pkt[14] = byte(cl >> 8)
	pkt[15] = byte(cl)
	if isLast {
		pkt[16] = 1
	}
	enc, err := encryptPacket(pkt)
	if err != nil {
		return fmt.Errorf("writeFileChunk encrypt: %w", err)
	}
	payload := make([]byte, len(enc)+len(chunk))
	copy(payload, enc)
	copy(payload[len(enc):], chunk)
	if _, err := d.epOut.Write(payload); err != nil {
		return fmt.Errorf("writeFileChunk write: %w", err)
	}
	resp := make([]byte, 512)
	d.epIn.Read(resp) //nolint:errcheck
	d.flush()
	return nil
}

// UploadFile uploads localData to remotePath on the device (cmd 38 + cmd 39 chunks).
func (d *Driver) UploadFile(remotePath string, localData []byte) error {
	d.log.Infof("TURZX: uploading %d B → %s", len(localData), remotePath)

	d.mu.Lock()
	err := d.openFile(remotePath)
	d.mu.Unlock()
	if err != nil {
		return err
	}

	total := len(localData)
	for offset := 0; offset < total; {
		end := offset + uploadChunkSize
		if end > total {
			end = total
		}
		chunk := localData[offset:end]
		isLast := end >= total

		d.mu.Lock()
		err := d.writeFileChunk(chunk, isLast)
		d.mu.Unlock()
		if err != nil {
			return fmt.Errorf("UploadFile chunk @%d: %w", offset, err)
		}
		offset = end
	}
	d.log.Infof("TURZX: upload complete (%d chunks)", (total+uploadChunkSize-1)/uploadChunkSize)
	return nil
}

// DeleteFile deletes a file from device storage (cmd 40).
func (d *Driver) DeleteFile(remotePath string) error {
	pkt := buildPathPacket(cmdDeleteFileUSB, remotePath)
	enc, err := encryptPacket(pkt)
	if err != nil {
		return fmt.Errorf("DeleteFile encrypt: %w", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, err := d.epOut.Write(enc); err != nil {
		return fmt.Errorf("DeleteFile write: %w", err)
	}
	buf := make([]byte, 512)
	d.epIn.Read(buf) //nolint:errcheck
	d.flush()
	return nil
}

// PlayVideoFromStorage requests playback of a stored video file (cmd 98).
func (d *Driver) PlayVideoFromStorage(remotePath string) error {
	return d.sendPathCmd(cmdPlayVideoUSB, remotePath)
}

// PlayVideoFromStorage2 is the alternate play command (cmd 110).
func (d *Driver) PlayVideoFromStorage2(remotePath string) error {
	return d.sendPathCmd(cmdPlayVideo2, remotePath)
}

// PlayImageFromStorage requests image display from device storage (cmd 113).
func (d *Driver) PlayImageFromStorage(remotePath string) error {
	return d.sendPathCmd(cmdPlayImage, remotePath)
}

// sendPathCmd sends any path-based command (cmds 38/40/98/110/113 all share the same layout).
func (d *Driver) sendPathCmd(cmdID int, remotePath string) error {
	pkt := buildPathPacket(cmdID, remotePath)
	enc, err := encryptPacket(pkt)
	if err != nil {
		return fmt.Errorf("cmd%d encrypt: %w", cmdID, err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, err := d.epOut.Write(enc); err != nil {
		return fmt.Errorf("cmd%d write: %w", cmdID, err)
	}
	buf := make([]byte, 512)
	d.epIn.Read(buf) //nolint:errcheck
	d.flush()
	return nil
}
