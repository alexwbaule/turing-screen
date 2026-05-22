package command

import (
	"math/big"
	"testing"

	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	pdevice "github.com/alexwbaule/turing-screen/internal/resource/process/device"
)

// mockImagePartial implements device.ImagePartial for testing
type mockImagePartial struct {
	data []byte
}

func (m *mockImagePartial) GeneratePartialImage(_ theme.Orientation, _ *device.Display, _, _ int, _ pdevice.PixelEncoding) []byte {
	return m.data
}

func (m *mockImagePartial) GetDimensions() (width, height int) {
	return 100, 50
}

func TestGetBytes_HeaderLayout(t *testing.T) {
	// Create an UpdatePayload with known payload and count
	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	up := &UpdatePayload{
		name:    "UPDATE_BITMAP",
		bytes:   []byte{0xcc, 0xef, 0x69, 0x00},
		padding: 0x00,
		size:    1024,
		payload: payload,
		count:   42,
	}

	chunks := up.GetBytes()
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	header := chunks[0]

	// Verify prefix bytes [0:4]
	if header[0] != 0xCC || header[1] != 0xEF || header[2] != 0x69 || header[3] != 0x00 {
		t.Errorf("prefix mismatch: got [%#x, %#x, %#x, %#x], want [0xCC, 0xEF, 0x69, 0x00]",
			header[0], header[1], header[2], header[3])
	}

	// Verify 3-byte size field at offset [4:7]
	expectedSize := len(payload) // 5
	sizeBytes := header[4:7]
	sizeVal := int(sizeBytes[0])<<16 | int(sizeBytes[1])<<8 | int(sizeBytes[2])
	if sizeVal != expectedSize {
		t.Errorf("size field: got %d, want %d (bytes: %#v)", sizeVal, expectedSize, sizeBytes)
	}

	// Verify padding at offset [7:10]
	if header[7] != 0x00 || header[8] != 0x00 || header[9] != 0x00 {
		t.Errorf("padding mismatch: got [%#x, %#x, %#x], want [0x00, 0x00, 0x00]",
			header[7], header[8], header[9])
	}

	// Verify 4-byte count field at offset [10:14]
	countBytes := header[10:14]
	countVal := int64(countBytes[0])<<24 | int64(countBytes[1])<<16 | int64(countBytes[2])<<8 | int64(countBytes[3])
	if countVal != 42 {
		t.Errorf("count field: got %d, want 42 (bytes: %#v)", countVal, countBytes)
	}
}

func TestGetBytes_LargeSize(t *testing.T) {
	// Test with a size that requires all 3 bytes
	payloadSize := 100000
	payload := make([]byte, payloadSize)

	up := &UpdatePayload{
		name:    "UPDATE_BITMAP",
		bytes:   []byte{0xcc, 0xef, 0x69, 0x00},
		padding: 0x00,
		size:    1024,
		payload: payload,
		count:   1,
	}

	chunks := up.GetBytes()
	header := chunks[0]

	// Verify 3-byte size field encodes 100000 correctly
	sizeBytes := header[4:7]
	sizeVal := int(sizeBytes[0])<<16 | int(sizeBytes[1])<<8 | int(sizeBytes[2])
	if sizeVal != payloadSize {
		t.Errorf("size field: got %d, want %d", sizeVal, payloadSize)
	}

	// Verify using big.Int for comparison
	expected := big.NewInt(int64(payloadSize)).Bytes()
	// expected should be 3 bytes for 100000: [0x01, 0x86, 0xA0]
	if len(expected) > 3 {
		t.Fatalf("test assumption failed: expected size fits in 3 bytes")
	}
}

func TestGetBytes_ZeroCount(t *testing.T) {
	up := &UpdatePayload{
		name:    "UPDATE_BITMAP",
		bytes:   []byte{0xcc, 0xef, 0x69, 0x00},
		padding: 0x00,
		size:    1024,
		payload: []byte{0xAA},
		count:   0,
	}

	chunks := up.GetBytes()
	header := chunks[0]

	// Count should be 4 zero bytes at offset [10:14]
	for i := 10; i < 14; i++ {
		if header[i] != 0x00 {
			t.Errorf("count byte at offset %d: got %#x, want 0x00", i, header[i])
		}
	}
}

func TestSendPayload_PayloadTooLarge(t *testing.T) {
	up := &UpdatePayload{
		log: nil,
	}

	// Create a mock partial that returns a payload larger than 16,777,215
	largePayload := make([]byte, 16_777_216)
	mock := &mockImagePartial{data: largePayload}

	_, err := up.SendPayload(mock, 0, 0, EncodingBGR)
	if err == nil {
		t.Fatal("expected error for payload > 16,777,215 bytes")
	}
	if err != ErrPayloadTooLarge {
		t.Errorf("expected ErrPayloadTooLarge, got: %v", err)
	}
}

func TestSendPayload_ValidPayload(t *testing.T) {
	up := &UpdatePayload{
		log: nil,
	}

	mock := &mockImagePartial{data: []byte{0x01, 0x02, 0x03}}

	result, err := up.SendPayload(mock, 10, 20, EncodingBGR)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.name != "UPDATE_BITMAP" {
		t.Errorf("name: got %q, want %q", result.name, "UPDATE_BITMAP")
	}
	// Verify the bytes prefix is 4 bytes
	if len(result.bytes) != 4 {
		t.Errorf("bytes length: got %d, want 4", len(result.bytes))
	}
	if result.bytes[0] != 0xcc || result.bytes[1] != 0xef || result.bytes[2] != 0x69 || result.bytes[3] != 0x00 {
		t.Errorf("bytes prefix mismatch: got %#v", result.bytes)
	}
}

func TestSendPayload_MaxValidSize(t *testing.T) {
	up := &UpdatePayload{
		log: nil,
	}

	// Exactly at the limit should succeed
	maxPayload := make([]byte, 16_777_215)
	mock := &mockImagePartial{data: maxPayload}

	_, err := up.SendPayload(mock, 0, 0, EncodingBGRA)
	if err != nil {
		t.Fatalf("unexpected error for max valid payload: %v", err)
	}
}
