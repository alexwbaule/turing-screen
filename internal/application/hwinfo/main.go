// Package hwinfo detects hardware model names for use as template variables
// in theme static_texts. Supported placeholders:
//
//	{{CPU_MODEL}}   - CPU model short name (e.g. "Ryzen 9 5900X")
//	{{GPU_MODEL}}   - GPU model short name (e.g. "RX 6800 XT")
//	{{DISK_MODEL}}  - Primary disk model (e.g. "Samsung 980 PRO 1TB")
//	{{MEM_TOTAL}}   - Total RAM formatted (e.g. "32 GB")
//	{{HOSTNAME}}    - System hostname
package hwinfo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	amdgpu "github.com/alexwbaule/turing-screen/internal/resource/gpu"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// HWInfo holds detected hardware model names.
type HWInfo struct {
	CPUModel  string
	GPUModel  string
	DiskModel string
	MemTotal  string
	Hostname  string
}

// Detect gathers hardware information from the system.
func Detect(log *logger.Logger, gpuProvider string) *HWInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info := &HWInfo{}

	// CPU Model
	cpuInfo, err := cpu.InfoWithContext(ctx)
	if err == nil && len(cpuInfo) > 0 {
		info.CPUModel = shortenCPUModel(cpuInfo[0].ModelName)
	} else {
		info.CPUModel = "Unknown CPU"
		log.Warnf("hwinfo: could not detect CPU model: %v", err)
	}

	// GPU Model
	info.GPUModel = shortenGPUModel(detectGPU(log, gpuProvider))

	// Disk Model
	info.DiskModel = shortenDiskModel(detectDisk(log))

	// Memory Total
	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		info.MemTotal = formatBytes(memInfo.Total)
	} else {
		info.MemTotal = "Unknown"
		log.Warnf("hwinfo: could not detect memory: %v", err)
	}

	// Hostname
	hostname, err := os.Hostname()
	if err == nil {
		info.Hostname = hostname
	} else {
		info.Hostname = "unknown"
	}

	log.Infof("hwinfo: CPU=%s, GPU=%s, Disk=%s, Mem=%s, Host=%s",
		info.CPUModel, info.GPUModel, info.DiskModel, info.MemTotal, info.Hostname)

	return info
}

// Replacements returns a map of placeholder → value for template substitution.
func (h *HWInfo) Replacements() map[string]string {
	return map[string]string{
		"{{CPU_MODEL}}":  h.CPUModel,
		"{{GPU_MODEL}}":  h.GPUModel,
		"{{DISK_MODEL}}": h.DiskModel,
		"{{MEM_TOTAL}}":  h.MemTotal,
		"{{HOSTNAME}}":   h.Hostname,
	}
}

// ReplaceText substitutes all known placeholders in the given text.
func (h *HWInfo) ReplaceText(text string) string {
	for placeholder, value := range h.Replacements() {
		text = strings.ReplaceAll(text, placeholder, value)
	}
	return text
}

// shortenCPUModel extracts the short model name from a full CPU model string.
// Examples:
//
//	"AMD Ryzen 9 5900X 12-Core Processor" → "Ryzen 9 5900X"
//	"AMD Ryzen 7 7800X3D 8-Core Processor" → "Ryzen 7 7800X3D"
//	"Intel(R) Core(TM) i9-10900K CPU @ 3.70GHz" → "i9-10900K"
//	"Intel(R) Core(TM) i7-12700K CPU @ 3.60GHz" → "i7-12700K"
//	"13th Gen Intel(R) Core(TM) i9-13900K" → "i9-13900K"
//	"AMD Ryzen Threadripper 3990X 64-Core Processor" → "Threadripper 3990X"
func shortenCPUModel(full string) string {
	full = strings.TrimSpace(full)
	if full == "" {
		return "Unknown CPU"
	}

	// Intel pattern: extract iX-XXXXX
	intelRe := regexp.MustCompile(`(i[3579]-\d{4,5}\w*)`)
	if m := intelRe.FindString(full); m != "" {
		return m
	}

	// AMD Ryzen pattern: "Ryzen X XXXXXX"
	ryzenRe := regexp.MustCompile(`(Ryzen\s+\d+\s+\w+)`)
	if m := ryzenRe.FindString(full); m != "" {
		return m
	}

	// AMD Threadripper pattern
	threadripperRe := regexp.MustCompile(`(Threadripper\s+\w+)`)
	if m := threadripperRe.FindString(full); m != "" {
		return m
	}

	// Fallback: remove common prefixes/suffixes
	result := full
	result = strings.ReplaceAll(result, "AMD ", "")
	result = strings.ReplaceAll(result, "Intel(R) ", "")
	result = strings.ReplaceAll(result, "Core(TM) ", "")
	result = strings.ReplaceAll(result, " CPU", "")
	result = strings.ReplaceAll(result, " Processor", "")

	// Remove frequency suffix like "@ 3.70GHz"
	if idx := strings.Index(result, " @"); idx > 0 {
		result = result[:idx]
	}

	// Remove core count suffix like "12-Core"
	coreRe := regexp.MustCompile(`\s+\d+-Core`)
	result = coreRe.ReplaceAllString(result, "")

	return strings.TrimSpace(result)
}

// shortenGPUModel extracts the short model name from a full GPU model string.
// Examples:
//
//	"AMD Radeon RX 6800 XT" → "RX 6800 XT"
//	"AMD Radeon RX 7900 XTX" → "RX 7900 XTX"
//	"NVIDIA GeForce RTX 4090" → "RTX 4090"
//	"NVIDIA GeForce GTX 1080 Ti" → "GTX 1080 Ti"
//	"AMD Radeon Pro W6800" → "Radeon Pro W6800"
func shortenGPUModel(full string) string {
	full = strings.TrimSpace(full)
	if full == "" || full == "Unknown GPU" || full == "AMD GPU" {
		return full
	}

	// NVIDIA: extract RTX/GTX XXXX [Ti/Super]
	nvidiaRe := regexp.MustCompile(`((?:RTX|GTX)\s+\d{4}\s*(?:Ti|Super|SUPER)?)`)
	if m := nvidiaRe.FindString(full); m != "" {
		return strings.TrimSpace(m)
	}

	// AMD RX: extract "RX XXXX [XT/XTX]"
	amdRxRe := regexp.MustCompile(`(RX\s+\d{4}\s*\w*)`)
	if m := amdRxRe.FindString(full); m != "" {
		return strings.TrimSpace(m)
	}

	// AMD Radeon Pro: keep "Radeon Pro XXXXX"
	radeonProRe := regexp.MustCompile(`(Radeon\s+Pro\s+\w+)`)
	if m := radeonProRe.FindString(full); m != "" {
		return m
	}

	// Fallback: remove common prefixes
	result := full
	result = strings.ReplaceAll(result, "AMD ", "")
	result = strings.ReplaceAll(result, "NVIDIA ", "")
	result = strings.ReplaceAll(result, "GeForce ", "")
	result = strings.ReplaceAll(result, "Radeon ", "")

	return strings.TrimSpace(result)
}

// shortenDiskModel extracts a cleaner model name from raw disk model strings.
// Examples:
//
//	"KINGSTON SKC3000S1024G" → "SKC3000S 1TB"
//	"Samsung SSD 980 PRO 1TB" → "980 PRO 1TB"
//	"Samsung SSD 870 EVO 500GB" → "870 EVO 500GB"
//	"WDC WD10EZEX-08WN4A0" → "WD10EZEX"
//	"CT1000MX500SSD1" → "MX500 1TB"
//	"INTEL SSDPEKNW512G8" → "SSDPEKNW 512GB"
//	"WD_BLACK SN850X 2000GB" → "SN850X 2TB"
func shortenDiskModel(full string) string {
	full = strings.TrimSpace(full)
	if full == "" || full == "Unknown Disk" {
		return full
	}

	// Remove common manufacturer prefixes
	prefixes := []string{
		"KINGSTON ", "Kingston ", "SAMSUNG ", "Samsung ", "samsung ",
		"WDC ", "WD_BLACK ", "INTEL ", "Intel ", "CRUCIAL ", "Crucial ",
		"ADATA ", "SK HYNIX ", "Sk Hynix ", "TOSHIBA ", "Toshiba ",
		"SEAGATE ", "Seagate ", "HITACHI ", "Hitachi ", "SANDISK ", "SanDisk ",
		"MICRON ", "Micron ", "PLEXTOR ", "Plextor ", "CORSAIR ", "Corsair ",
		"SABRENT ", "Sabrent ", "PNY ", "TEAM ", "Team ",
	}
	result := full
	for _, prefix := range prefixes {
		if strings.HasPrefix(result, prefix) {
			result = strings.TrimPrefix(result, prefix)
			break
		}
	}

	// Remove "SSD " prefix if still present
	result = strings.TrimPrefix(result, "SSD ")

	// Remove trailing part numbers after dash (e.g. "WD10EZEX-08WN4A0" → "WD10EZEX")
	if idx := strings.Index(result, "-"); idx > 4 {
		// Only trim if the part after dash looks like a part number (alphanumeric)
		after := result[idx+1:]
		if len(after) > 3 && isAlphaNum(after) {
			result = result[:idx]
		}
	}

	// Try to extract and humanize capacity from model string
	result = humanizeDiskCapacity(result)

	return strings.TrimSpace(result)
}

// humanizeDiskCapacity finds capacity patterns in disk model names and formats them.
// "SKC3000S1024G" → "SKC3000S 1TB"
// "SSDPEKNW512G8" → "SSDPEKNW 512GB"
// "500GB" stays as "500GB"
// "2000GB" → "2TB"
func humanizeDiskCapacity(model string) string {
	// Pattern: digits followed by G or GB or T or TB at the end or before a single digit suffix
	capRe := regexp.MustCompile(`(\d{3,5})(G)\d?$`)
	if m := capRe.FindStringSubmatchIndex(model); m != nil {
		prefix := strings.TrimSpace(model[:m[2]])
		sizeStr := model[m[2]:m[3]]
		size := 0
		fmt.Sscanf(sizeStr, "%d", &size)

		var capacity string
		if size >= 1000 {
			tb := size / 1000
			capacity = fmt.Sprintf("%dTB", tb)
		} else {
			capacity = fmt.Sprintf("%dGB", size)
		}

		if prefix != "" {
			return prefix + " " + capacity
		}
		return capacity
	}

	// Pattern: "2000GB" → "2TB"
	gbRe := regexp.MustCompile(`(\d{4,5})\s*GB`)
	if m := gbRe.FindStringSubmatch(model); len(m) > 1 {
		size := 0
		fmt.Sscanf(m[1], "%d", &size)
		if size >= 1000 {
			tb := size / 1000
			model = gbRe.ReplaceAllString(model, fmt.Sprintf("%dTB", tb))
		}
	}

	return model
}

func isAlphaNum(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}

func detectGPU(log *logger.Logger, provider string) string {
	// Try AMD first
	if provider == "auto" || provider == "amd" {
		cards := amdgpu.GetAMDGPUs()
		if len(cards) > 0 {
			// Try sysfs product_name (not always available)
			name := readSysfs("/sys/class/drm/" + cards[0] + "/device/product_name")
			if name != "" {
				return name
			}
			name = readSysfs("/sys/class/drm/card0/device/product_name")
			if name != "" {
				return name
			}

			// Fallback: use lspci to get the marketing name from PCI ID
			vendorID := readSysfs("/sys/class/drm/" + cards[0] + "/device/vendor")
			deviceID := readSysfs("/sys/class/drm/" + cards[0] + "/device/device")
			if vendorID == "" || deviceID == "" {
				vendorID = readSysfs("/sys/class/drm/card0/device/vendor")
				deviceID = readSysfs("/sys/class/drm/card0/device/device")
			}
			if vendorID != "" && deviceID != "" {
				name = detectGPUFromLspci(log, vendorID, deviceID)
				if name != "" {
					return name
				}
			}

			return "AMD GPU"
		}
	}

	// Try NVIDIA
	if provider == "auto" || provider == "nvidia" {
		out, err := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader").Output()
		if err == nil {
			name := strings.TrimSpace(string(out))
			if name != "" {
				return name
			}
		}
	}

	return "Unknown GPU"
}

// detectGPUFromLspci uses lspci to find the GPU marketing name by PCI vendor:device ID.
// vendorID and deviceID are in "0x1002" format from sysfs.
func detectGPUFromLspci(log *logger.Logger, vendorID, deviceID string) string {
	// Strip "0x" prefix for lspci -d format: "1002:73df"
	vid := strings.TrimPrefix(vendorID, "0x")
	did := strings.TrimPrefix(deviceID, "0x")
	pciID := vid + ":" + did

	out, err := exec.Command("lspci", "-d", pciID).Output()
	if err != nil {
		log.Debugf("hwinfo: lspci failed for %s: %v", pciID, err)
		return ""
	}

	line := strings.TrimSpace(string(out))
	if line == "" {
		return ""
	}

	// lspci output format: "0c:00.0 VGA compatible controller: Advanced Micro Devices, Inc. [AMD/ATI] Navi 22 [Radeon RX 6700/6700 XT...]"
	// Extract the part inside the last square brackets which contains the marketing name
	lastBracket := strings.LastIndex(line, "[")
	if lastBracket > 0 && strings.HasSuffix(line, "]") {
		name := line[lastBracket+1 : len(line)-1]
		// May contain multiple models like "Radeon RX 6700/6700 XT/6750 XT / 6800M/6850M XT"
		// Take the first model listed
		if idx := strings.Index(name, "/"); idx > 0 {
			// Check if it's "RX 6700/6700 XT" pattern — keep the first complete model
			name = strings.TrimSpace(name[:idx])
			// If it ends with a number, it's likely incomplete (e.g. "Radeon RX 6700")
			// That's still fine as a display name
		}
		return strings.TrimSpace(name)
	}

	// Fallback: extract after the colon that follows the device class
	parts := strings.SplitN(line, ": ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}

	return ""
}

func detectDisk(log *logger.Logger) string {
	// Try to get the model of the root filesystem's disk
	partitions, err := disk.Partitions(false)
	if err != nil || len(partitions) == 0 {
		return "Unknown Disk"
	}

	// Find the root partition
	for _, p := range partitions {
		if p.Mountpoint == "/" {
			// Extract device name (e.g. /dev/nvme0n1p2 → nvme0n1)
			devName := extractDiskDevice(p.Device)
			if devName != "" {
				model := readSysfs("/sys/block/" + devName + "/device/model")
				if model != "" {
					return strings.TrimSpace(model)
				}
				// For NVMe, try a different path
				model = readSysfs("/sys/block/" + devName + "/device/subsystem_model")
				if model != "" {
					return strings.TrimSpace(model)
				}
			}
		}
	}

	return "Unknown Disk"
}

func extractDiskDevice(devPath string) string {
	// /dev/nvme0n1p2 → nvme0n1
	// /dev/sda1 → sda
	name := devPath
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}

	// Strip partition number
	// NVMe: nvme0n1p2 → nvme0n1
	if strings.Contains(name, "nvme") {
		if idx := strings.LastIndex(name, "p"); idx > 0 {
			// Make sure what follows 'p' is a digit
			rest := name[idx+1:]
			if len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
				return name[:idx]
			}
		}
		return name
	}

	// SATA/SCSI: sda1 → sda
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] < '0' || name[i] > '9' {
			return name[:i+1]
		}
	}
	return name
}

func readSysfs(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	sizes := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.0f %s", float64(b)/float64(div), sizes[exp])
}
