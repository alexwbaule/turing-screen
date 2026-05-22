package sensors

import (
	"context"
	"strings"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/shirou/gopsutil/v3/host"
)

// findSensorTemperature searches for a temperature sensor by configured name or auto-detection patterns.
// It returns the temperature value and the critical threshold percentage.
//
// Logic:
// 1. If configuredName is set, look for an exact match among available sensors.
//   - If found, return its temperature and percent of critical.
//   - If not found, log a warning and return 0, 0.
//
// 2. If configuredName is empty, attempt auto-detection using the provided patterns.
//   - Select the first sensor whose key contains any of the patterns (case-insensitive).
//   - If no match, log a warning and return 0, 0.
func findSensorTemperature(ctx context.Context, configuredName string, autoDetectPatterns []string, log *logger.Logger) (temperature float64, percent float64) {
	stats, err := host.SensorsTemperaturesWithContext(ctx)
	if err != nil {
		log.Warnf("failed to read temperature sensors: %v", err)
		return 0, 0
	}

	if configuredName != "" {
		// Exact match for configured sensor name
		for _, stat := range stats {
			if stat.SensorKey == configuredName {
				temperature = stat.Temperature
				if stat.Critical > 0 {
					percent = (stat.Temperature / stat.Critical) * 100
				}
				return temperature, percent
			}
		}
		log.Warnf("configured sensor %q not found in available sensors, reporting zero", configuredName)
		return 0, 0
	}

	// Auto-detection: find first sensor matching any pattern
	for _, stat := range stats {
		keyLower := strings.ToLower(stat.SensorKey)
		for _, pattern := range autoDetectPatterns {
			if strings.Contains(keyLower, strings.ToLower(pattern)) {
				log.Debugf("auto-detected temperature sensor: %s", stat.SensorKey)
				temperature = stat.Temperature
				if stat.Critical > 0 {
					percent = (stat.Temperature / stat.Critical) * 100
				}
				return temperature, percent
			}
		}
	}

	log.Warnf("auto-detection found no matching sensor for patterns %v, reporting zero", autoDetectPatterns)
	return 0, 0
}
