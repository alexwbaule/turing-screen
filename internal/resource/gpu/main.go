package gpu

import (
	"os/exec"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/resource/gpu/amd"
	nvidia "github.com/alexwbaule/turing-screen/internal/resource/gpu/nividia"
	"github.com/alexwbaule/turing-screen/internal/resource/gpu/noop"
	"github.com/alexwbaule/turing-screen/internal/resource/interfaces"
)

// GPUMetrics holds the metrics collected from a GPU.

// NewGPUProvider creates a GPUProvider based on the provider string.
// Supported values: "auto", "amd", "nvidia", "none".
// "auto" detects available GPU, preferring AMD if both are present.
func NewGPUProvider(name string, log *logger.Logger) interfaces.Provider {
	switch name {
	case "amd":
		provider := amd.NewAMDProvider(log)
		if provider == nil {
			return noop.NewNoopProvider(log)
		}
		return provider
	case "nvidia":
		provider := nvidia.NewNvidiaProvider(log)
		if provider == nil {
			return noop.NewNoopProvider(log)
		}
	case "none":
		return noop.NewNoopProvider(log)
	case "auto":
		return autoDetectProvider(log)
	default:
		log.Warnf("unknown GPU provider %q, falling back to auto-detection", name)
		return autoDetectProvider(log)
	}
	return noop.NewNoopProvider(log)
}

// autoDetectProvider checks for available GPUs, preferring AMD if present.
func autoDetectProvider(log *logger.Logger) interfaces.Provider {
	// Check AMD first (preferred)
	cards := amd.GetAMDGPUs()
	if len(cards) > 0 {
		log.Infof("auto-detected AMD GPU: %s", cards[0])
		return amd.NewAMDProvider(log)
	}

	// Check NVIDIA via nvidia-smi
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		log.Info("auto-detected NVIDIA GPU via nvidia-smi")
		return nvidia.NewNvidiaProvider(log)
	}

	// No GPU found
	log.Warn("no supported GPU detected, using noop provider")
	return noop.NewNoopProvider(log)
}
