package amdgpu

import (
	"testing"
)

func TestParseNvidiaSMIOutput_ValidInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *GPUMetrics
	}{
		{
			name:  "typical output with spaces",
			input: "65, 42, 120.50, 4096, 8192\n",
			expected: &GPUMetrics{
				Temperature: 65,
				Load:        42,
				Power:       121, // 120.50 rounds to 121
				VRAMUsage:   4096 * 1024 * 1024,
				VRAMSize:    8192 * 1024 * 1024,
			},
		},
		{
			name:  "no spaces between fields",
			input: "70,100,250.00,16384,24576",
			expected: &GPUMetrics{
				Temperature: 70,
				Load:        100,
				Power:       250,
				VRAMUsage:   16384 * 1024 * 1024,
				VRAMSize:    24576 * 1024 * 1024,
			},
		},
		{
			name:  "zero values",
			input: "0, 0, 0.00, 0, 0\n",
			expected: &GPUMetrics{
				Temperature: 0,
				Load:        0,
				Power:       0,
				VRAMUsage:   0,
				VRAMSize:    0,
			},
		},
		{
			name:  "fractional power value",
			input: "85, 99, 349.75, 12000, 12288\n",
			expected: &GPUMetrics{
				Temperature: 85,
				Load:        99,
				Power:       350, // 349.75 rounds to 350
				VRAMUsage:   12000 * 1024 * 1024,
				VRAMSize:    12288 * 1024 * 1024,
			},
		},
		{
			name:  "trailing whitespace and newline",
			input: "  55 , 30 , 75.25 , 2048 , 4096  \n",
			expected: &GPUMetrics{
				Temperature: 55,
				Load:        30,
				Power:       75, // 75.25 rounds to 75
				VRAMUsage:   2048 * 1024 * 1024,
				VRAMSize:    4096 * 1024 * 1024,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseNvidiaSMIOutput(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Temperature != tt.expected.Temperature {
				t.Errorf("Temperature: got %d, want %d", result.Temperature, tt.expected.Temperature)
			}
			if result.Load != tt.expected.Load {
				t.Errorf("Load: got %d, want %d", result.Load, tt.expected.Load)
			}
			if result.Power != tt.expected.Power {
				t.Errorf("Power: got %d, want %d", result.Power, tt.expected.Power)
			}
			if result.VRAMUsage != tt.expected.VRAMUsage {
				t.Errorf("VRAMUsage: got %d, want %d", result.VRAMUsage, tt.expected.VRAMUsage)
			}
			if result.VRAMSize != tt.expected.VRAMSize {
				t.Errorf("VRAMSize: got %d, want %d", result.VRAMSize, tt.expected.VRAMSize)
			}
		})
	}
}

func TestParseNvidiaSMIOutput_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "whitespace only",
			input: "   \n",
		},
		{
			name:  "too few fields",
			input: "65, 42, 120.50",
		},
		{
			name:  "too many fields",
			input: "65, 42, 120.50, 4096, 8192, 99",
		},
		{
			name:  "non-numeric temperature",
			input: "abc, 42, 120.50, 4096, 8192",
		},
		{
			name:  "non-numeric utilization",
			input: "65, xyz, 120.50, 4096, 8192",
		},
		{
			name:  "non-numeric power",
			input: "65, 42, bad, 4096, 8192",
		},
		{
			name:  "non-numeric memory used",
			input: "65, 42, 120.50, N/A, 8192",
		},
		{
			name:  "non-numeric memory total",
			input: "65, 42, 120.50, 4096, N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseNvidiaSMIOutput(tt.input)
			if err == nil {
				t.Fatalf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

func TestParseNvidiaSMIOutput_MemoryConversion(t *testing.T) {
	// Verify MiB to bytes conversion: 1 MiB = 1048576 bytes
	input := "50, 25, 100.00, 1, 1\n"
	result, err := parseNvidiaSMIOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.VRAMUsage != 1048576 {
		t.Errorf("VRAMUsage: got %d, want %d (1 MiB in bytes)", result.VRAMUsage, 1048576)
	}
	if result.VRAMSize != 1048576 {
		t.Errorf("VRAMSize: got %d, want %d (1 MiB in bytes)", result.VRAMSize, 1048576)
	}
}
