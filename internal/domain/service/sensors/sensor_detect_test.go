package sensors

import (
	"context"
	"testing"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
)

func TestFindSensorTemperature_AutoDetection(t *testing.T) {
	// This test exercises the auto-detection path on the current system.
	// It won't fail if no sensors are found (returns 0, 0 with a warning).
	log := logger.NewLogger()
	ctx := context.Background()

	// CPU auto-detection patterns
	cpuPatterns := []string{"cpu", "tdie"}
	temp, pct := findSensorTemperature(ctx, "", cpuPatterns, log)
	// We can't assert specific values since they depend on hardware,
	// but we can verify the function doesn't panic and returns valid floats.
	if temp < 0 || pct < 0 {
		t.Errorf("expected non-negative values, got temp=%f, pct=%f", temp, pct)
	}

	// Disk auto-detection patterns
	diskPatterns := []string{"nvme", "disk"}
	temp, pct = findSensorTemperature(ctx, "", diskPatterns, log)
	if temp < 0 || pct < 0 {
		t.Errorf("expected non-negative values, got temp=%f, pct=%f", temp, pct)
	}
}

func TestFindSensorTemperature_ConfiguredNotFound(t *testing.T) {
	// When a configured sensor name doesn't exist, should return 0, 0.
	log := logger.NewLogger()
	ctx := context.Background()

	temp, pct := findSensorTemperature(ctx, "nonexistent_sensor_xyz_12345", nil, log)
	if temp != 0 || pct != 0 {
		t.Errorf("expected (0, 0) for nonexistent sensor, got (%f, %f)", temp, pct)
	}
}

func TestFindSensorTemperature_NoMatchingPatterns(t *testing.T) {
	// When auto-detection patterns don't match anything, should return 0, 0.
	log := logger.NewLogger()
	ctx := context.Background()

	patterns := []string{"zzz_nonexistent_pattern_xyz"}
	temp, pct := findSensorTemperature(ctx, "", patterns, log)
	if temp != 0 || pct != 0 {
		t.Errorf("expected (0, 0) for unmatched patterns, got (%f, %f)", temp, pct)
	}
}
