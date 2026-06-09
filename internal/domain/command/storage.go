package command

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/utils"
)

const (
	cmdGetStorageStatus = 0x64
	cmdListDir          = 0x65
	cmdDeleteFile       = 0x66
	cmdGetFileInfo      = 0x6e
	cmdCreateFile       = 0x6f
	cmdPlayVideo        = 0x78
	cmdSetStartMode     = 0x7b
	cmdRestartDevice    = 0x82
)

type Padding []byte

var (
	playVideoSuccess                     = regexp.MustCompile(`^play_video_success$`)
	storageStatus                        = regexp.MustCompile(`^\d+-\d+-\d+-\d+-\d+-\d+$`)
	listDirResult                        = regexp.MustCompile(`^result:dir:`)
	deleteSuccess                        = regexp.MustCompile(`^delete_success$`)
	createSuccess                        = regexp.MustCompile(`^create_success$`)
	fileRevDone                          = regexp.MustCompile(`^file_rev_done`)
	fileSizeResponse                     = regexp.MustCompile(`^\d+$`)
	PADDING_NULL                 Padding = []byte{0x00}
	PADDING_START_DISPLAY_BITMAP Padding = []byte{0x2c}
)

// StorageInfo holds parsed storage status from the device.
type StorageInfo struct {
	TotalKB int
	UsedKB  int
	FreeKB  int
}

// ParseStorageInfo parses the device storage response "total-used-free-0-0-0".
func ParseStorageInfo(response []byte) (*StorageInfo, error) {
	raw := string(bytes.Trim(response, "\x00"))
	parts := strings.Split(raw, "-")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid storage response: %q", raw)
	}
	total, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid total in storage response: %w", err)
	}
	used, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid used in storage response: %w", err)
	}
	free, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid free in storage response: %w", err)
	}
	return &StorageInfo{TotalKB: total, UsedKB: used, FreeKB: free}, nil
}

// Storage provides commands for device storage operations.
type Storage struct {
	bytes     []byte
	name      string
	padding   byte
	payload   []byte
	size      int
	videoSize int64
	readed    *regexp.Regexp
	log       *logger.Logger
}

func NewStorage(log *logger.Logger) *Storage {
	return &Storage{
		log:       log,
		videoSize: 0,
	}
}

func (s *Storage) GetBytes() [][]byte {
	if s.name == "UPLOAD_FILE" {
		var chunks [][]byte
		dataSize := len(s.payload)

		for i := 0; i < dataSize; i += 249 {
			end := i + 249
			if end > dataSize {
				end = dataSize
			}

			packet := utils.BZero(250, s.padding)
			copy(packet, s.payload[i:end])
			chunks = append(chunks, packet)
		}
		return chunks
	}

	if s.bytes == nil {
		return nil
	}
	tmp := utils.BZero(250, s.padding)
	copy(tmp, s.bytes)
	return [][]byte{tmp}
}

func (s *Storage) SetCount(count int64) {
	_ = count
}

func (s *Storage) GetName() string {
	return s.name
}

func (s *Storage) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  s.size,
		Bytes: nil,
	}
}

func (s *Storage) ValidateCommand(data []byte, i int) error {
	if s.readed == nil {
		return nil
	}
	v := string(bytes.Trim(data[:i], "\x00"))
	if s.readed.MatchString(v) {
		if s.name == "GET_FILE_INFO" {
			bts, err := utils.ParseFileSize(data)
			if err != nil {
				return fmt.Errorf("unexpected response for %s: %q", s.name, v)
			}
			s.videoSize = bts
		}
		return nil
	}
	return fmt.Errorf("unexpected response for %s: %q", s.name, v)
}

func (s *Storage) GetVideoSize() int64 {
	time.Sleep(3 * time.Second)
	s.log.Debugf("Returning video size: %d", s.videoSize)
	return s.videoSize
}

// buildPathPayload creates the command payload for path-based commands.
// Format: [cmd:6][path_len:1][0x00:3][path:path_len]
func buildPathPayload(cmd []byte, path string) []byte {
	pyd := []byte{byte(len(path))}
	pyd = append(pyd, PADDING_NULL[0], PADDING_NULL[0], PADDING_NULL[0])
	pyd = append(pyd, []byte(path)...)

	finalCmd := make([]byte, 0, len(cmd)+len(pyd))
	finalCmd = append(finalCmd, cmd...)
	finalCmd = append(finalCmd, pyd...)

	return finalCmd
}

// GetStorageStatus queries the device storage capacity.
// Response: "total-used-free-0-0-0" (values in KB).
func (s *Storage) GetStorageStatus() *Storage {
	return &Storage{
		name: "GET_STORAGE_STATUS",
		bytes: []byte{
			cmdGetStorageStatus, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding:   0x00,
		size:      1024,
		readed:    storageStatus,
		log:       s.log,
		videoSize: 0,
	}
}

// ListDir lists files in a device directory.
// Response: "result:dir:file:name1/name2/..."
func (s *Storage) ListDir(path string) *Storage {
	return &Storage{
		name:    "LIST_DIR",
		bytes:   buildPathPayload([]byte{cmdListDir, 0xef, 0x69, 0x00, 0x00, 0x00}, path),
		padding: 0x00,
		size:    1024,
		readed:  listDirResult,
		log:     s.log,
	}
}

// DeleteFile deletes a file from the device storage.
// Response: "delete_success".
func (s *Storage) DeleteFile(path string) *Storage {
	return &Storage{
		name:    "DELETE_FILE",
		bytes:   buildPathPayload([]byte{cmdDeleteFile, 0xef, 0x69, 0x00, 0x00, 0x00}, utils.ParsePath(path)),
		padding: 0x00,
		size:    1024,
		readed:  deleteSuccess,
		log:     s.log,
	}
}

// GetFileInfo queries the size of a file on the device.
// Response: ASCII decimal file size (e.g. "480610").
// Returns "0" if the file does not exist.
func (s *Storage) GetFileInfo(path string) *Storage {
	return &Storage{
		name:    "GET_FILE_INFO",
		bytes:   buildPathPayload([]byte{cmdGetFileInfo, 0xef, 0x69, 0x00, 0x00, 0x00}, utils.ParsePath(path)),
		padding: 0x00,
		size:    1024,
		readed:  fileSizeResponse,
		log:     s.log,
	}
}

// CreateFile initiates a file upload to the device.
// The command includes the destination path and the file size (little-endian).
// Response: "create_success".
// After receiving "create_success", the caller must send the raw file bytes
// in chunks, then wait for "file_rev_done" (or "file_rev_doneimg_show_").
func (s *Storage) CreateFile(path string, fileSize int64) *Storage {
	return &Storage{
		name:    "CREATE_FILE",
		bytes:   buildCreateFilePayload(utils.ParsePath(path), fileSize),
		padding: 0x00,
		size:    1024,
		readed:  createSuccess,
		log:     s.log,
	}
}

// UploadFile prepares the command to send the file content in chunks.
func (s *Storage) UploadFile(data []byte) *Storage {
	return &Storage{
		name:    "UPLOAD_FILE",
		payload: data,
		padding: 0x00,
		size:    1024,
		readed:  fileRevDone,
		log:     s.log,
	}
}

// buildCreateFilePayload creates the CREATE_FILE command payload.
// Format: [cmd:6][path_len:1][0x00:3][path:path_len][file_size:4LE]
func buildCreateFilePayload(path string, fileSize int64) []byte {
	cmd := []byte{0x6f, 0xef, 0x69, 0x00, 0x00, 0x00}

	pyd := []byte{byte(len(path))}
	pyd = append(pyd, PADDING_NULL[0], PADDING_NULL[0], PADDING_NULL[0])
	pyd = append(pyd, []byte(path)...)

	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, uint32(fileSize))
	pyd = append(pyd, sizeBuf...)

	finalCmd := make([]byte, 0, len(cmd)+len(pyd))
	finalCmd = append(finalCmd, cmd...)
	finalCmd = append(finalCmd, pyd...)

	return finalCmd
}

// PlayVideo starts playback of a video file from device storage.
// The loop parameter controls whether the video loops (true) or plays once (false).
// Response: "play_video_success".
func (s *Storage) PlayVideo(path string, loop bool) *Storage {
	return &Storage{
		name:    "PLAY_VIDEO",
		bytes:   buildPlayVideoPayload(path, loop),
		padding: 0x00,
		size:    1024,
		readed:  playVideoSuccess,
		log:     s.log,
	}
}

// buildPlayVideoPayload creates the PLAY_VIDEO command payload.
// Format: [0x78][0xEF69][path_len:4BE][loop_flag:1][0x0000][path]
// The loop flag byte (byte[7]) is 0x01 for loop, 0x00 for play-once.
func buildPlayVideoPayload(path string, loop bool) []byte {
	pathBytes := []byte(path)
	pathLen := len(pathBytes)

	payload := make([]byte, 0, 10+pathLen)
	payload = append(payload, cmdPlayVideo, 0xef, 0x69)
	// path length as 4 bytes big-endian
	payload = append(payload, byte(pathLen>>24), byte(pathLen>>16), byte(pathLen>>8), byte(pathLen))
	// loop flag + 2 null bytes (separator)
	if loop {
		payload = append(payload, 0x01, 0x00, 0x00)
	} else {
		payload = append(payload, 0x00, 0x00, 0x00)
	}
	// path
	payload = append(payload, pathBytes...)

	return payload
}

// SetStartModeDefault sends the SET_START_MODE command with default mode (0x8a00).
// This is sent before RESTART when preparing for video playback.
// No response expected.
func (s *Storage) SetStartModeDefault() *Storage {
	return &Storage{
		name: "SET_START_MODE_DEFAULT",
		bytes: []byte{
			cmdSetStartMode, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x8a, 0x00,
		},
		padding: 0x00,
		size:    0,
		readed:  nil,
		log:     s.log,
	}
}

// SetStartModeVideo sends the SET_START_MODE command with video mode (0x8a02).
// No response expected.
func (s *Storage) SetStartModeVideo() *Storage {
	return &Storage{
		name: "SET_START_MODE_VIDEO",
		bytes: []byte{
			cmdSetStartMode, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x8a, 0x02,
		},
		padding: 0x00,
		size:    0,
		readed:  nil,
		log:     s.log,
	}
}

// RestartDevice sends the device restart command (0x82).
// This is different from the full hardware RESTART (0x84) - it's a soft restart
// used in the video playback sequence.
// No response expected.
func (s *Storage) RestartDevice() *Storage {
	return &Storage{
		name: "RESTART_DEVICE",
		bytes: []byte{
			cmdRestartDevice, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		size:    0,
		readed:  nil,
		log:     s.log,
	}
}

// SetPreUpload sends CMD_0x7D which prepares the device for file upload.
// Must be sent before CREATE_FILE. The value byte matches the brightness setting.
// No response expected.
func (s *Storage) SetPreUpload(brightness byte) *Storage {
	return &Storage{
		name: "SET_PRE_UPLOAD",
		bytes: []byte{
			0x7d, 0xef, 0x69, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, brightness, 0x00,
		},
		padding: 0x00,
		size:    0,
		readed:  nil,
		log:     s.log,
	}
}
