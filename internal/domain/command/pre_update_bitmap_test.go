package command

import (
	"testing"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
)

func TestPreUpdateBitmap_GetBytes(t *testing.T) {
	log := logger.NewLogger()
	cmd := NewPreUpdateBitmap(log)

	chunks := cmd.GetBytes()
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]
	if len(chunk) != 250 {
		t.Fatalf("expected chunk length 250, got %d", len(chunk))
	}

	expected := []byte{0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01}
	for i, b := range expected {
		if chunk[i] != b {
			t.Errorf("byte %d: expected 0x%02x, got 0x%02x", i, b, chunk[i])
		}
	}

	// Remaining bytes should be zero padding
	for i := len(expected); i < 250; i++ {
		if chunk[i] != 0x00 {
			t.Errorf("padding byte %d: expected 0x00, got 0x%02x", i, chunk[i])
		}
	}
}

func TestPreUpdateBitmap_GetName(t *testing.T) {
	log := logger.NewLogger()
	cmd := NewPreUpdateBitmap(log)

	if cmd.GetName() != "PRE_UPDATE_BITMAP" {
		t.Errorf("expected name PRE_UPDATE_BITMAP, got %s", cmd.GetName())
	}
}

func TestPreUpdateBitmap_ImplementsCommand(t *testing.T) {
	log := logger.NewLogger()
	var _ Command = NewPreUpdateBitmap(log)
}
