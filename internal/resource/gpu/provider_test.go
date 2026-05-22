package amdgpu

import (
	"testing"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
)

func TestNoopProvider_GetMetrics_ReturnsZero(t *testing.T) {
	log := logger.NewLogger()
	p := &noopProvider{log: log}

	metrics, err := p.GetMetrics()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if metrics.Temperature != 0 {
		t.Errorf("expected Temperature=0, got %d", metrics.Temperature)
	}
	if metrics.Load != 0 {
		t.Errorf("expected Load=0, got %d", metrics.Load)
	}
	if metrics.Power != 0 {
		t.Errorf("expected Power=0, got %d", metrics.Power)
	}
	if metrics.VRAMUsage != 0 {
		t.Errorf("expected VRAMUsage=0, got %d", metrics.VRAMUsage)
	}
	if metrics.VRAMSize != 0 {
		t.Errorf("expected VRAMSize=0, got %d", metrics.VRAMSize)
	}
}

func TestNoopProvider_Available_ReturnsFalse(t *testing.T) {
	log := logger.NewLogger()
	p := &noopProvider{log: log}

	if p.Available() {
		t.Error("expected Available()=false for noopProvider")
	}
}

func TestNewGPUProvider_None_ReturnsNoopProvider(t *testing.T) {
	log := logger.NewLogger()
	p := NewGPUProvider("none", log)

	if p.Available() {
		t.Error("expected noop provider to report Available()=false")
	}

	metrics, err := p.GetMetrics()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if metrics.Temperature != 0 || metrics.Load != 0 || metrics.Power != 0 {
		t.Errorf("expected zero metrics from noop provider, got %+v", metrics)
	}
}

func TestNewGPUProvider_UnknownFallsBackToAuto(t *testing.T) {
	log := logger.NewLogger()
	// Unknown provider should fall back to auto-detection without panicking
	p := NewGPUProvider("unknown_provider", log)
	if p == nil {
		t.Fatal("expected non-nil provider for unknown provider string")
	}
}

func TestNewGPUProvider_NvidiaMismatch_FallsBackToNoop(t *testing.T) {
	// On a system without nvidia-smi, requesting "nvidia" should fall back to noop
	log := logger.NewLogger()
	p := NewGPUProvider("nvidia", log)

	// Since this test runs in CI/dev without nvidia-smi, it should fall back to noop
	// The provider should not panic and should return zero metrics
	metrics, err := p.GetMetrics()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}

	// Verify it fell back to noop (not available)
	if p.Available() {
		t.Skip("nvidia-smi is available on this system, skipping mismatch test")
	}
}

func TestNewGPUProvider_AmdMismatch_FallsBackToNoop(t *testing.T) {
	// This test verifies the mismatch detection logic.
	// If no AMD GPU is present, requesting "amd" should fall back to noop.
	// If an AMD GPU IS present, the provider should be an amdProvider.
	log := logger.NewLogger()
	p := NewGPUProvider("amd", log)

	// The provider should not panic regardless of hardware
	if p == nil {
		t.Fatal("expected non-nil provider")
	}

	cards := GetAMDGPUs()
	if len(cards) == 0 {
		// No AMD GPU: should have fallen back to noop
		if p.Available() {
			t.Error("expected Available()=false when no AMD GPU present")
		}
		metrics, err := p.GetMetrics()
		if err != nil {
			t.Fatalf("expected no error from noop fallback, got %v", err)
		}
		if metrics.Temperature != 0 || metrics.Load != 0 {
			t.Errorf("expected zero metrics from noop fallback, got %+v", metrics)
		}
	} else {
		// AMD GPU present: should be an amdProvider
		if !p.Available() {
			t.Error("expected Available()=true when AMD GPU is present")
		}
	}
}

func TestNewGPUProvider_Auto_ReturnsProvider(t *testing.T) {
	log := logger.NewLogger()
	p := NewGPUProvider("auto", log)

	if p == nil {
		t.Fatal("expected non-nil provider for auto detection")
	}

	// Should not panic regardless of hardware.
	// On systems with GPU hardware that can't be initialized (permissions),
	// GetMetrics may return an error — that's acceptable.
	_, _ = p.GetMetrics()
}
