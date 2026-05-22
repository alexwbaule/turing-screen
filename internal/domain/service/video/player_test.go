package video

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
)

func newTestLogger() *logger.Logger {
	return logger.NewLogger()
}

func TestValidateLocal_FileNotFound(t *testing.T) {
	vp := &VideoPlayer{log: newTestLogger()}
	err := vp.validateLocal("/nonexistent/video.mp4")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestValidateLocal_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")
	os.WriteFile(path, make([]byte, 1024), 0644)

	vp := &VideoPlayer{log: newTestLogger()}
	err := vp.validateLocal(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestNewVideoPlayer(t *testing.T) {
	log := newTestLogger()
	storage := command.NewStorage(log)
	vp := NewVideoPlayer(nil, log, storage)
	if vp == nil {
		t.Fatal("expected non-nil player")
	}
}

func TestPlayCommands_NoVideos(t *testing.T) {
	log := newTestLogger()
	storage := command.NewStorage(log)
	vp := NewVideoPlayer(nil, log, storage)

	cmds, _, err := vp.PlayCommands()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmds != nil {
		t.Fatal("expected nil commands")
	}
}

func TestPlayCommands_FileNotFound(t *testing.T) {
	log := newTestLogger()
	storage := command.NewStorage(log)
	videos := map[string]theme.DinamicImage{
		"bg": {Path: "/nonexistent.mp4", Width: 800, Height: 480},
	}
	vp := NewVideoPlayer(videos, log, storage)

	// PlayCommands does not validate the local file — it only builds
	// the command sequence assuming the video already exists on the device.
	// Local file validation is the caller's responsibility.
	cmds, devicePath, err := vp.PlayCommands()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmds == nil {
		t.Fatal("expected non-nil commands")
	}
	if devicePath != "/root/video/nonexistent.mp4" {
		t.Fatalf("expected /root/video/nonexistent.mp4, got %s", devicePath)
	}
}

func TestPlayCommands_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")
	os.WriteFile(path, []byte("fake"), 0644)

	log := newTestLogger()
	storage := command.NewStorage(log)
	videos := map[string]theme.DinamicImage{
		"bg": {Path: path, Width: 800, Height: 480},
	}
	vp := NewVideoPlayer(videos, log, storage)

	cmds, devicePath, err := vp.PlayCommands()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if devicePath != "/root/video/test.mp4" {
		t.Fatalf("expected /root/video/test.mp4, got %s", devicePath)
	}

	// GET_STORAGE_STATUS, GET_FILE_INFO, RESTART_DEVICE, GET_FILE_INFO, PLAY_VIDEO
	expected := []string{"GET_STORAGE_STATUS", "GET_FILE_INFO", "RESTART_DEVICE", "GET_FILE_INFO", "PLAY_VIDEO"}
	if len(cmds) != len(expected) {
		t.Fatalf("expected %d commands, got %d", len(expected), len(cmds))
	}
	for i, cmd := range cmds {
		if cmd.GetName() != expected[i] {
			t.Errorf("cmd %d: expected %s, got %s", i, expected[i], cmd.GetName())
		}
	}
}

func TestUploadCommands_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")
	os.WriteFile(path, []byte("fake mp4 data"), 0644)

	log := newTestLogger()
	storage := command.NewStorage(log)
	videos := map[string]theme.DinamicImage{
		"bg": {Path: path, Width: 800, Height: 480},
	}
	vp := NewVideoPlayer(videos, log, storage)

	cmds, err := vp.UploadCommands("/root/video/test.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// GET_STORAGE_STATUS, DELETE_FILE, GET_STORAGE_STATUS, CREATE_FILE, UPLOAD_DATA, UPLOAD_DONE
	expected := []string{"GET_STORAGE_STATUS", "DELETE_FILE", "GET_STORAGE_STATUS", "CREATE_FILE", "UPLOAD_DATA", "UPLOAD_DONE"}
	if len(cmds) != len(expected) {
		t.Fatalf("expected %d commands, got %d", len(expected), len(cmds))
	}
	for i, cmd := range cmds {
		if cmd.GetName() != expected[i] {
			t.Errorf("cmd %d: expected %s, got %s", i, expected[i], cmd.GetName())
		}
	}
}

func TestUploadDataCommand_Chunks(t *testing.T) {
	data := make([]byte, 150000)
	cmd := &uploadDataCommand{data: data, name: "UPLOAD_DATA"}

	chunks := cmd.GetBytes()
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 65536 {
		t.Fatalf("chunk 0: expected 65536, got %d", len(chunks[0]))
	}
	if len(chunks[2]) != 18928 {
		t.Fatalf("chunk 2: expected 18928, got %d", len(chunks[2]))
	}
}
