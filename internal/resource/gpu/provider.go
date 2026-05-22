package amdgpu

import (
	"os/exec"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
)

// GPUMetrics holds the metrics collected from a GPU.
type GPUMetrics struct {
	Temperature uint64 // °C
	Load        uint64 // percentage
	Power       uint64 // watts
	VRAMUsage   uint64 // bytes
	VRAMSize    uint64 // bytes
}

// GPUProvider is the interface for reading GPU metrics from any vendor.
type GPUProvider interface {
	// GetMetrics returns the current GPU metrics.
	GetMetrics() (*GPUMetrics, error)
	// Available reports whether the GPU provider has a usable GPU.
	Available() bool
}

// NewGPUProvider creates a GPUProvider based on the provider string.
// Supported values: "auto", "amd", "nvidia", "none".
// "auto" detects available GPU, preferring AMD if both are present.
func NewGPUProvider(provider string, log *logger.Logger) GPUProvider {
	switch provider {
	case "amd":
		return newAMDProvider(log)
	case "nvidia":
		return newNvidiaProvider(log)
	case "none":
		return newNoopProvider(log)
	case "auto":
		return autoDetectProvider(log)
	default:
		log.Warnf("unknown GPU provider %q, falling back to auto-detection", provider)
		return autoDetectProvider(log)
	}
}

// autoDetectProvider checks for available GPUs, preferring AMD if present.
func autoDetectProvider(log *logger.Logger) GPUProvider {
	// Check AMD first (preferred)
	cards := GetAMDGPUs()
	if len(cards) > 0 {
		log.Infof("auto-detected AMD GPU: %s", cards[0])
		return newAMDProvider(log)
	}

	// Check NVIDIA via nvidia-smi
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		log.Info("auto-detected NVIDIA GPU via nvidia-smi")
		return newNvidiaProvider(log)
	}

	// No GPU found
	log.Warn("no supported GPU detected, using noop provider")
	return newNoopProvider(log)
}

// newAMDProvider creates an AMD GPU provider wrapping existing amdgpu functions.
// If no AMD GPU is found, it falls back to noop with a warning (Req 14.6).
func newAMDProvider(log *logger.Logger) GPUProvider {
	cards := GetAMDGPUs()
	if len(cards) == 0 {
		log.Warn("GPU provider configured as \"amd\" but no AMD GPU found, falling back to noop")
		return newNoopProvider(log)
	}
	return &amdProvider{log: log}
}

// amdProvider wraps the existing amdgpu package functions into the GPUProvider interface.
type amdProvider struct {
	log *logger.Logger
}

func (p *amdProvider) GetMetrics() (*GPUMetrics, error) {
	cards := GetAMDGPUs()
	if len(cards) == 0 {
		p.log.Warn("no AMD GPU found")
		return &GPUMetrics{}, nil
	}

	sensors, err := GetCardSensor(cards[0])
	if err != nil {
		return nil, err
	}

	return &GPUMetrics{
		Temperature: sensors["GPU_TEMP"],
		Load:        sensors["GPU_LOAD"],
		Power:       sensors["GPU_AVG_POWER"],
		VRAMUsage:   sensors["VRAM_USAGE"],
		VRAMSize:    sensors["VRAM_SIZE"],
	}, nil
}

func (p *amdProvider) Available() bool {
	cards := GetAMDGPUs()
	return len(cards) > 0
}

// newNoopProvider creates a noop GPU provider that returns zero metrics.
// Logs a warning at startup indicating no GPU metrics will be reported (Req 14.3).
func newNoopProvider(log *logger.Logger) GPUProvider {
	log.Warn("GPU provider set to noop: all GPU metrics will report zero values")
	return &noopProvider{log: log}
}

// noopProvider returns zero metrics for all GPU fields.
type noopProvider struct {
	log *logger.Logger
}

func (p *noopProvider) GetMetrics() (*GPUMetrics, error) {
	return &GPUMetrics{}, nil
}

func (p *noopProvider) Available() bool {
	return false
}
