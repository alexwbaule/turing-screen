package amdgpu

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
)

const (
	nvidiaSMITimeout = 5 * time.Second
	nvidiaSMIBinary  = "nvidia-smi"
	nvidiaSMIArgs    = "--query-gpu=temperature.gpu,utilization.gpu,power.draw,memory.used,memory.total"
	nvidiaSMIFormat  = "--format=csv,noheader,nounits"
)

// newNvidiaProvider creates an NVIDIA GPU provider.
// If nvidia-smi is not available, it falls back to noop with a warning (Req 14.6).
func newNvidiaProvider(log *logger.Logger) GPUProvider {
	if _, err := exec.LookPath(nvidiaSMIBinary); err != nil {
		log.Warn("GPU provider configured as \"nvidia\" but nvidia-smi not found in PATH, falling back to noop")
		return newNoopProvider(log)
	}
	return &nvidiaProvider{log: log}
}

// nvidiaProvider reads GPU metrics via nvidia-smi.
type nvidiaProvider struct {
	log *logger.Logger
}

func (p *nvidiaProvider) GetMetrics() (*GPUMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), nvidiaSMITimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, nvidiaSMIBinary, nvidiaSMIArgs, nvidiaSMIFormat)
	output, err := cmd.Output()
	if err != nil {
		p.log.Errorf("nvidia-smi failed: %v", err)
		return &GPUMetrics{}, nil
	}

	metrics, err := parseNvidiaSMIOutput(string(output))
	if err != nil {
		p.log.Errorf("failed to parse nvidia-smi output: %v", err)
		return &GPUMetrics{}, nil
	}

	return metrics, nil
}

func (p *nvidiaProvider) Available() bool {
	_, err := exec.LookPath(nvidiaSMIBinary)
	return err == nil
}

// parseNvidiaSMIOutput parses the CSV output from nvidia-smi into GPUMetrics.
// Expected format: "65, 42, 120.50, 4096, 8192\n"
// Fields: temperature(°C), utilization(%), power(W), memory_used(MiB), memory_total(MiB)
func parseNvidiaSMIOutput(output string) (*GPUMetrics, error) {
	line := strings.TrimSpace(output)
	if line == "" {
		return nil, fmt.Errorf("empty nvidia-smi output")
	}

	fields := strings.Split(line, ",")
	if len(fields) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d: %q", len(fields), line)
	}

	// Parse temperature (integer °C)
	temp, err := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse temperature %q: %w", fields[0], err)
	}

	// Parse utilization (integer %)
	util, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse utilization %q: %w", fields[1], err)
	}

	// Parse power draw (float watts)
	power, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse power %q: %w", fields[2], err)
	}

	// Parse memory used (MiB, convert to bytes)
	memUsed, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory used %q: %w", fields[3], err)
	}

	// Parse memory total (MiB, convert to bytes)
	memTotal, err := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory total %q: %w", fields[4], err)
	}

	return &GPUMetrics{
		Temperature: uint64(math.Round(temp)),
		Load:        uint64(math.Round(util)),
		Power:       uint64(math.Round(power)),
		VRAMUsage:   uint64(math.Round(memUsed)) * 1024 * 1024,  // MiB to bytes
		VRAMSize:    uint64(math.Round(memTotal)) * 1024 * 1024, // MiB to bytes
	}, nil
}
