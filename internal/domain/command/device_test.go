package command

import (
	"testing"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
)

func TestParseHelloResponse_ValidROM88(t *testing.T) {
	log := logger.NewLogger()
	response := []byte("chs_5inch.dev1_rom1.88\x00\x00\x00")

	result, err := ParseHelloResponse(response, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ROMVersion != 88 {
		t.Errorf("expected ROM version 88, got %d", result.ROMVersion)
	}
	if result.RawString != "chs_5inch.dev1_rom1.88" {
		t.Errorf("expected raw string 'chs_5inch.dev1_rom1.88', got %q", result.RawString)
	}
}

func TestParseHelloResponse_ValidROM90(t *testing.T) {
	log := logger.NewLogger()
	response := []byte("chs_5inch.dev1_rom1.90\x00\x00\x00")

	result, err := ParseHelloResponse(response, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ROMVersion != 90 {
		t.Errorf("expected ROM version 90, got %d", result.ROMVersion)
	}
}

func TestParseHelloResponse_InvalidResponse_DefaultsTo99(t *testing.T) {
	log := logger.NewLogger()
	response := []byte("unknown_device_response\x00\x00")

	result, err := ParseHelloResponse(response, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ROMVersion != 99 {
		t.Errorf("expected default ROM version 99, got %d", result.ROMVersion)
	}
	if result.RawString != "unknown_device_response" {
		t.Errorf("expected raw string 'unknown_device_response', got %q", result.RawString)
	}
}

func TestParseHelloResponse_EmptyResponse_DefaultsTo99(t *testing.T) {
	log := logger.NewLogger()
	response := []byte("\x00\x00\x00")

	result, err := ParseHelloResponse(response, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ROMVersion != 99 {
		t.Errorf("expected default ROM version 99, got %d", result.ROMVersion)
	}
}

func TestParseHelloResponse_ValidROM00(t *testing.T) {
	log := logger.NewLogger()
	response := []byte("chs_5inch.dev1_rom1.00\x00")

	result, err := ParseHelloResponse(response, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ROMVersion != 0 {
		t.Errorf("expected ROM version 0, got %d", result.ROMVersion)
	}
}
