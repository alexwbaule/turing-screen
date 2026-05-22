package video

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
)

const (
	// maxFileSize is the maximum allowed video file size in bytes (10 MB).
	maxFileSize = 10 * 1024 * 1024
	// deviceVideoDir is the path on the device where videos are stored.
	deviceVideoDir = "/root/video/"
)

// VideoPlayer builds the sequence of commands to play a video on the device.
// Commands are enqueued into the normal worker queue — same serial channel,
// same sequential command-response flow as everything else.
type VideoPlayer struct {
	videos  map[string]theme.DinamicImage
	log     *logger.Logger
	storage *command.Storage
}

// NewVideoPlayer creates a new VideoPlayer.
func NewVideoPlayer(
	videos map[string]theme.DinamicImage,
	log *logger.Logger,
	storage *command.Storage,
) *VideoPlayer {
	return &VideoPlayer{
		videos:  videos,
		log:     log.With("component", "video_player"),
		storage: storage,
	}
}

// PlayCommands returns the commands to play a video that is already on the device.
// Sequence (validated with real hardware via play_video.py):
//
//	GET_STORAGE_STATUS → storage info
//	GET_FILE_INFO(path) → file size (verifies file exists)
//	RESTART_DEVICE (0x82)
//	(caller inserts HELLO here)
//	GET_FILE_INFO(path) → file size (verify again)
//	PLAY_VIDEO(path, loop=true) → "play_video_success"
//
// The caller must insert a HELLO command after RESTART_DEVICE.
func (v *VideoPlayer) PlayCommands() ([]command.Command, string, error) {
	if len(v.videos) == 0 {
		return nil, "", nil
	}

	for name, vid := range v.videos {
		localPath := vid.Path
		v.log.Infof("preparing video: %s (%s)", name, localPath)

		// Extract filename — the video must already exist on the device.
		// No local file validation needed (upload is a separate operation).
		fileName := filepath.Base(localPath)
		devicePath := deviceVideoDir + fileName
		v.log.Infof("device path: %s", devicePath)

		var cmds []command.Command

		// GET_STORAGE_STATUS
		cmds = append(cmds, v.storage.GetStorageStatus())

		// GET_FILE_INFO — verify file exists on device
		cmds = append(cmds, v.storage.GetFileInfo(devicePath))

		// RESTART_DEVICE (soft restart 0x82)
		cmds = append(cmds, v.storage.RestartDevice())

		// (caller inserts HELLO here)

		// GET_FILE_INFO — verify again after restart
		cmds = append(cmds, v.storage.GetFileInfo(devicePath))

		// PLAY_VIDEO with loop=true
		cmds = append(cmds, v.storage.PlayVideo(devicePath, true))

		return cmds, devicePath, nil
	}

	return nil, "", nil
}

// DevicePath returns the device path for the first video entry.
func (v *VideoPlayer) DevicePath() string {
	for _, vid := range v.videos {
		fileName := filepath.Base(vid.Path)
		return deviceVideoDir + fileName
	}
	return ""
}

// LocalPath returns the local file path for the first video entry.
func (v *VideoPlayer) LocalPath() string {
	for _, vid := range v.videos {
		return vid.Path
	}
	return ""
}

// UploadCommands returns the commands to upload a video file to the device.
// This is only needed if the file doesn't exist on the device yet.
// Sequence (from BlueTheme-Flip180.txt PCAP):
//
//	GET_STORAGE_STATUS → verify space
//	DELETE_FILE(path) → "delete_success" (cleanup old)
//	GET_STORAGE_STATUS → verify space after delete
//	CREATE_FILE(path, size) → "create_success"
//	[raw file data]
//	← "file_rev_done"
func (v *VideoPlayer) UploadCommands(devicePath string) ([]command.Command, error) {
	if len(v.videos) == 0 {
		return nil, nil
	}

	for _, vid := range v.videos {
		localPath := vid.Path

		data, err := os.ReadFile(localPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read video file: %w", err)
		}

		fileSize := int64(len(data))
		v.log.Infof("upload: %s (%d bytes) -> %s", localPath, fileSize, devicePath)

		var cmds []command.Command

		// GET_STORAGE_STATUS
		cmds = append(cmds, v.storage.GetStorageStatus())

		// DELETE old file (ignore errors if not exists)
		cmds = append(cmds, v.storage.DeleteFile(devicePath))

		// GET_STORAGE_STATUS (verify space)
		cmds = append(cmds, v.storage.GetStorageStatus())

		// CREATE_FILE
		cmds = append(cmds, v.storage.CreateFile(devicePath, fileSize))

		// Raw file data
		cmds = append(cmds, &uploadDataCommand{data: data, name: "UPLOAD_DATA"})

		// Wait for "file_rev_done"
		cmds = append(cmds, v.storage.UploadDone())

		return cmds, nil
	}

	return nil, nil
}

// validateLocal checks that the local video file exists and is within size limits.
func (v *VideoPlayer) validateLocal(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("video file not found: %s", path)
		}
		return fmt.Errorf("cannot stat video file: %w", err)
	}

	if info.Size() > maxFileSize {
		return fmt.Errorf("video file exceeds %d MB size limit: %d bytes", maxFileSize/(1024*1024), info.Size())
	}

	return nil
}

// uploadDataCommand carries raw file bytes to be written to the serial port.
// No framing, no response — just raw data sent in chunks.
type uploadDataCommand struct {
	data []byte
	name string
}

func (u *uploadDataCommand) GetBytes() [][]byte {
	const chunkSize = 65536
	var chunks [][]byte
	for i := 0; i < len(u.data); i += chunkSize {
		end := i + chunkSize
		if end > len(u.data) {
			end = len(u.data)
		}
		chunks = append(chunks, u.data[i:end])
	}
	return chunks
}

func (u *uploadDataCommand) GetName() string                        { return u.name }
func (u *uploadDataCommand) ValidateWrite() command.WriteValidation { return command.WriteValidation{} }
func (u *uploadDataCommand) ValidateCommand([]byte, int) error      { return nil }
func (u *uploadDataCommand) SetCount(int64)                         {}
