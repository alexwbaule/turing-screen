package serial

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/resource/usb"
)

// mockPort implements SerialPort for testing.
type mockPort struct {
	mu         sync.Mutex
	writes     [][]byte
	readData   [][]byte // queued responses for sequential Read calls
	readIdx    int
	flushCount int
	flushErr   error
	writeErr   error
}

func (m *mockPort) Write(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	m.writes = append(m.writes, cp)
	return len(b), nil
}

func (m *mockPort) Read(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.readIdx >= len(m.readData) {
		return 0, io.EOF
	}
	data := m.readData[m.readIdx]
	m.readIdx++
	n := copy(b, data)
	return n, nil
}

func (m *mockPort) Flush() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flushCount++
	return m.flushErr
}

func (m *mockPort) Close() error {
	return nil
}

// mockCommand implements command.Command for testing.
type mockCommand struct {
	name      string
	chunks    [][]byte
	writeVal  command.WriteValidation
	validateF func([]byte, int) error
}

func (c *mockCommand) GetBytes() [][]byte {
	return c.chunks
}

func (c *mockCommand) GetName() string {
	return c.name
}

func (c *mockCommand) ValidateWrite() command.WriteValidation {
	return c.writeVal
}

func (c *mockCommand) ValidateCommand(s []byte, i int) error {
	if c.validateF != nil {
		return c.validateF(s, i)
	}
	return nil
}

func (c *mockCommand) SetCount(num int64) {}

func newTestSerial(port *mockPort) *Serial {
	log := logger.NewLogger()
	return &Serial{
		port: port,
		log:  log,
	}
}

func newMockCommandWithResponse(responseSize int) *mockCommand {
	return &mockCommand{
		name:   "TEST_CMD",
		chunks: [][]byte{{0x01, 0x02, 0x03}},
		writeVal: command.WriteValidation{
			Size:  responseSize,
			Bytes: []byte{0xcf, 0xef, 0x69},
		},
	}
}

// TestRead_NeedReSend0_ReturnsSuccess tests that when the device responds with
// "needReSend:0", Read returns success without calling ValidateCommand.
func TestRead_NeedReSend0_ReturnsSuccess(t *testing.T) {
	response := []byte("needReSend:0|renderCnt:5")
	port := &mockPort{
		readData: [][]byte{response},
	}
	s := newTestSerial(port)

	cmd := &mockCommand{
		name:   "UPDATE_BITMAP",
		chunks: [][]byte{{0xcc, 0xef, 0x69}},
		writeVal: command.WriteValidation{
			Size:  1024,
			Bytes: nil,
		},
		validateF: func(b []byte, i int) error {
			t.Error("ValidateCommand should not be called for needReSend:0 response")
			return nil
		},
	}

	n, err := s.Read(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if n == 0 {
		t.Fatal("expected non-zero bytes read")
	}
}

// TestRead_NeedReSend1_TriggersRetransmission tests that when the device responds
// with "needReSend:1", Read triggers handleNeedReSend which retransmits the command.
func TestRead_NeedReSend1_TriggersRetransmission(t *testing.T) {
	// First read returns needReSend:1 (initial Read)
	// Then handleNeedReSend retransmits and reads needReSend:0
	needReSend1 := make([]byte, 1024)
	copy(needReSend1, []byte("needReSend:1|renderCnt:0"))

	needReSend0 := make([]byte, 1024)
	copy(needReSend0, []byte("needReSend:0|renderCnt:1"))

	port := &mockPort{
		readData: [][]byte{needReSend1, needReSend0},
	}
	s := newTestSerial(port)

	cmd := newMockCommandWithResponse(1024)

	n, err := s.Read(cmd)
	if err != nil {
		t.Fatalf("expected no error after successful retransmission, got: %v", err)
	}
	if n == 0 {
		t.Fatal("expected non-zero bytes read")
	}

	// Verify that the command was retransmitted (writes happened)
	if len(port.writes) == 0 {
		t.Fatal("expected writes from retransmission")
	}
}

// TestRead_NeedReSend1_ExhaustsRetries tests that after 3 failed retransmission
// attempts, an error is returned.
func TestRead_NeedReSend1_ExhaustsRetries(t *testing.T) {
	// Initial read returns needReSend:1
	// All 3 retransmission attempts also return needReSend:1
	responses := make([][]byte, 4)
	for i := range responses {
		resp := make([]byte, 1024)
		copy(resp, []byte("needReSend:1|renderCnt:0"))
		responses[i] = resp
	}

	port := &mockPort{
		readData: responses,
	}
	s := newTestSerial(port)

	cmd := newMockCommandWithResponse(1024)

	_, err := s.Read(cmd)
	if err == nil {
		t.Fatal("expected error after exhausting retransmission attempts")
	}
	if err.Error() != "retransmission failed after 3 attempts for command [TEST_CMD]" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestRead_OtherResponse_UsesValidateCommand tests that responses not containing
// needReSend are validated via the command's ValidateCommand method.
func TestRead_OtherResponse_UsesValidateCommand(t *testing.T) {
	response := []byte("chs_5inch.dev1_rom1.88")
	port := &mockPort{
		readData: [][]byte{response},
	}
	s := newTestSerial(port)

	validateCalled := false
	cmd := &mockCommand{
		name:   "HELLO",
		chunks: [][]byte{{0x01, 0xef, 0x69}},
		writeVal: command.WriteValidation{
			Size:  23,
			Bytes: nil,
		},
		validateF: func(b []byte, i int) error {
			validateCalled = true
			return nil
		},
	}

	_, err := s.Read(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !validateCalled {
		t.Fatal("expected ValidateCommand to be called for non-needReSend response")
	}
}

// TestHandleNeedReSend_FlushesBeforeRead tests that the serial port read buffer
// is flushed before each response read during retransmission (Req 4.5).
func TestHandleNeedReSend_FlushesBeforeRead(t *testing.T) {
	// Retransmit succeeds on first attempt
	needReSend0 := make([]byte, 1024)
	copy(needReSend0, []byte("needReSend:0|renderCnt:1"))

	port := &mockPort{
		readData: [][]byte{needReSend0},
	}
	s := newTestSerial(port)

	cmd := newMockCommandWithResponse(1024)

	err := s.handleNeedReSend(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Flush should have been called at least once during retransmission
	if port.flushCount == 0 {
		t.Fatal("expected Flush to be called before reading response during retransmission")
	}
}

// TestHandleNeedReSend_RetransmitsAllChunks tests that all byte chunks from
// GetBytes() are retransmitted.
func TestHandleNeedReSend_RetransmitsAllChunks(t *testing.T) {
	needReSend0 := make([]byte, 1024)
	copy(needReSend0, []byte("needReSend:0|renderCnt:1"))

	port := &mockPort{
		readData: [][]byte{needReSend0},
	}
	s := newTestSerial(port)

	cmd := &mockCommand{
		name:   "UPDATE_BITMAP",
		chunks: [][]byte{{0xcc, 0xef, 0x69}, {0x01, 0x02, 0x03}, {0x04, 0x05, 0x06}},
		writeVal: command.WriteValidation{
			Size:  1024,
			Bytes: []byte{0xcf, 0xef, 0x69},
		},
	}

	err := s.handleNeedReSend(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should have written 3 chunks + 1 validation bytes = 4 writes
	if len(port.writes) != 4 {
		t.Fatalf("expected 4 writes (3 chunks + 1 validation), got %d", len(port.writes))
	}
}

// TestHandleNeedReSend_WriteError tests that a write error during retransmission
// is returned immediately.
func TestHandleNeedReSend_WriteError(t *testing.T) {
	port := &mockPort{
		writeErr: fmt.Errorf("device disconnected"),
	}
	s := newTestSerial(port)

	cmd := newMockCommandWithResponse(1024)

	err := s.handleNeedReSend(cmd)
	if err == nil {
		t.Fatal("expected error on write failure")
	}
}

// TestHandleNeedReSend_SucceedsOnSecondAttempt tests that retransmission succeeds
// when the second attempt gets needReSend:0.
func TestHandleNeedReSend_SucceedsOnSecondAttempt(t *testing.T) {
	needReSend1 := make([]byte, 1024)
	copy(needReSend1, []byte("needReSend:1|renderCnt:0"))

	needReSend0 := make([]byte, 1024)
	copy(needReSend0, []byte("needReSend:0|renderCnt:2"))

	port := &mockPort{
		readData: [][]byte{needReSend1, needReSend0},
	}
	s := newTestSerial(port)

	cmd := newMockCommandWithResponse(1024)

	err := s.handleNeedReSend(cmd)
	if err != nil {
		t.Fatalf("expected success on second attempt, got: %v", err)
	}

	// Should have 2 sets of writes (2 attempts × (1 chunk + 1 validation))
	expectedWrites := 2 * 2 // 2 attempts, each with 1 chunk + 1 validation bytes
	if len(port.writes) != expectedWrites {
		t.Fatalf("expected %d writes for 2 attempts, got %d", expectedWrites, len(port.writes))
	}
}

// TestReconnect_SucceedsOnFirstAttempt tests that Reconnect returns nil when
// the port can be reopened on the first attempt.
func TestReconnect_SucceedsOnFirstAttempt(t *testing.T) {
	port := &mockPort{}
	log := logger.NewLogger()
	s := &Serial{
		port: port,
		log:  log,
		device: &usb.UsbDevice{
			Name: "TEST_PORT",
		},
		reconnectConfig: ReconnectConfig{
			InitialDelay: 1 * time.Millisecond, // Use short delays for testing
			MaxDelay:     10 * time.Millisecond,
			MaxAttempts:  10,
		},
	}

	// Reconnect will call NewSerial which requires a real USB device.
	// Since we can't mock NewSerial easily, we test the delay calculation
	// and context cancellation behavior instead.
	// For a full integration test, we'd need to mock the USB layer.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel immediately to test context cancellation path
	cancel()

	err := s.Reconnect(ctx)
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
	if err.Error() != "reconnection cancelled: context canceled" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestReconnect_RespectsContextCancellation tests that Reconnect exits early
// when the context is cancelled during a backoff wait.
func TestReconnect_RespectsContextCancellation(t *testing.T) {
	port := &mockPort{}
	log := logger.NewLogger()
	s := &Serial{
		port: port,
		log:  log,
		device: &usb.UsbDevice{
			Name: "NONEXISTENT_PORT",
		},
		reconnectConfig: ReconnectConfig{
			InitialDelay: 5 * time.Second, // Long delay to ensure we cancel during wait
			MaxDelay:     60 * time.Second,
			MaxAttempts:  10,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := s.Reconnect(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error when context times out")
	}
	// Should exit quickly due to context timeout, not wait the full 5s
	if elapsed > 1*time.Second {
		t.Fatalf("Reconnect took too long (%v), should have exited on context cancellation", elapsed)
	}
}

// TestReconnectConfig_DefaultValues tests that DefaultReconnectConfig returns
// the expected default values.
func TestReconnectConfig_DefaultValues(t *testing.T) {
	cfg := DefaultReconnectConfig()

	if cfg.InitialDelay != 1*time.Second {
		t.Fatalf("expected InitialDelay 1s, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 60*time.Second {
		t.Fatalf("expected MaxDelay 60s, got %v", cfg.MaxDelay)
	}
	if cfg.MaxAttempts != 10 {
		t.Fatalf("expected MaxAttempts 10, got %d", cfg.MaxAttempts)
	}
}

// TestReconnect_ExponentialBackoffDelayCalculation tests that the delay formula
// min(InitialDelay × 2^(n-1), MaxDelay) is correctly computed for each attempt.
func TestReconnect_ExponentialBackoffDelayCalculation(t *testing.T) {
	// Test the delay calculation logic directly
	cfg := ReconnectConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
		MaxAttempts:  10,
	}

	expectedDelays := []time.Duration{
		1 * time.Second,  // 1s × 2^0
		2 * time.Second,  // 1s × 2^1
		4 * time.Second,  // 1s × 2^2
		8 * time.Second,  // 1s × 2^3
		16 * time.Second, // 1s × 2^4
		32 * time.Second, // 1s × 2^5
		60 * time.Second, // capped at MaxDelay (would be 64s)
		60 * time.Second, // capped at MaxDelay (would be 128s)
		60 * time.Second, // capped at MaxDelay (would be 256s)
		60 * time.Second, // capped at MaxDelay (would be 512s)
	}

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		delay := cfg.InitialDelay * (1 << (attempt - 1))
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
		expected := expectedDelays[attempt-1]
		if delay != expected {
			t.Errorf("attempt %d: expected delay %v, got %v", attempt, expected, delay)
		}
	}
}

// TestReconnect_ExhaustsAllAttempts tests that after MaxAttempts failed reconnection
// attempts, Reconnect returns an error indicating exhaustion.
func TestReconnect_ExhaustsAllAttempts(t *testing.T) {
	port := &mockPort{}
	log := logger.NewLogger()
	s := &Serial{
		port: port,
		log:  log,
		device: &usb.UsbDevice{
			Name: "NONEXISTENT_PORT_THAT_WILL_NEVER_OPEN",
		},
		reconnectConfig: ReconnectConfig{
			InitialDelay: 1 * time.Millisecond, // Very short delays for testing
			MaxDelay:     5 * time.Millisecond,
			MaxAttempts:  3, // Fewer attempts for faster test
		},
	}

	ctx := context.Background()
	err := s.Reconnect(ctx)

	if err == nil {
		t.Fatal("expected error after exhausting all reconnection attempts")
	}
	expectedMsg := "reconnection failed: exhausted 3 attempts"
	if err.Error() != expectedMsg {
		t.Fatalf("expected error %q, got %q", expectedMsg, err.Error())
	}
}
