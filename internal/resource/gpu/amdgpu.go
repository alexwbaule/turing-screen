// Package amdgpu is a collection of utility functions to access various properties
// of AMD GPU via Linux kernel interfaces like sysfs and ioctl (using libdrm.)
package amdgpu

// #cgo pkg-config: libdrm libdrm_amdgpu
// #include <stdint.h>
// #include <xf86drm.h>
// #include <drm.h>
// #include <amdgpu.h>
// #include <amdgpu_drm.h>
import "C"
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

func GetCardSensor(cardName string) (map[string]uint64, error) {
	devHandle, err := openAMDGPU(cardName)
	if err != nil {
		return map[string]uint64{}, err
	}
	defer C.amdgpu_device_deinitialize(devHandle)

	var measurement C.uint32_t
	var measurement64 C.uint64_t
	var info C.struct_drm_amdgpu_info_vram_gtt

	featVersions := map[string]uint64{}

	C.amdgpu_query_sensor_info(devHandle, C.AMDGPU_INFO_SENSOR_GFX_SCLK, C.uint(unsafe.Sizeof(measurement)), unsafe.Pointer(&measurement))
	featVersions["GFX_SCLK"] = uint64(measurement)

	C.amdgpu_query_sensor_info(devHandle, C.AMDGPU_INFO_SENSOR_GFX_MCLK, C.uint(unsafe.Sizeof(measurement)), unsafe.Pointer(&measurement))
	featVersions["GFX_MCLK"] = uint64(measurement)

	C.amdgpu_query_sensor_info(devHandle, C.AMDGPU_INFO_SENSOR_GPU_TEMP, C.uint(unsafe.Sizeof(measurement)), unsafe.Pointer(&measurement))
	featVersions["GPU_TEMP"] = uint64(measurement) / 1000

	C.amdgpu_query_sensor_info(devHandle, C.AMDGPU_INFO_SENSOR_GPU_LOAD, C.uint(unsafe.Sizeof(measurement)), unsafe.Pointer(&measurement))
	featVersions["GPU_LOAD"] = uint64(measurement)

	C.amdgpu_query_sensor_info(devHandle, C.AMDGPU_INFO_SENSOR_GPU_AVG_POWER, C.uint(unsafe.Sizeof(measurement)), unsafe.Pointer(&measurement))
	featVersions["GPU_AVG_POWER"] = uint64(measurement)

	C.amdgpu_query_sensor_info(devHandle, C.AMDGPU_INFO_SENSOR_VDDNB, C.uint(unsafe.Sizeof(measurement)), unsafe.Pointer(&measurement))
	featVersions["VDDNB"] = uint64(measurement)

	C.amdgpu_query_sensor_info(devHandle, C.AMDGPU_INFO_SENSOR_VDDGFX, C.uint(unsafe.Sizeof(measurement)), unsafe.Pointer(&measurement))
	featVersions["VDDGFX"] = uint64(measurement)

	C.amdgpu_query_sensor_info(devHandle, C.AMDGPU_INFO_SENSOR_STABLE_PSTATE_GFX_SCLK, C.uint(unsafe.Sizeof(measurement)), unsafe.Pointer(&measurement))
	featVersions["STABLE_PSTATE_GFX_SCLK"] = uint64(measurement)

	C.amdgpu_query_sensor_info(devHandle, C.AMDGPU_INFO_SENSOR_STABLE_PSTATE_GFX_MCLK, C.uint(unsafe.Sizeof(measurement)), unsafe.Pointer(&measurement))
	featVersions["STABLE_PSTATE_GFX_MCLK"] = uint64(measurement)

	C.amdgpu_query_info(devHandle, C.AMDGPU_INFO_GTT_USAGE, C.uint(unsafe.Sizeof(measurement64)), unsafe.Pointer(&measurement64))
	featVersions["GTT_USAGE"] = uint64(measurement64)

	C.amdgpu_query_info(devHandle, C.AMDGPU_INFO_VRAM_USAGE, C.uint(unsafe.Sizeof(measurement64)), unsafe.Pointer(&measurement64))
	featVersions["VRAM_USAGE"] = uint64(measurement64)

	C.amdgpu_query_info(devHandle, C.AMDGPU_INFO_VRAM_USAGE, C.uint(unsafe.Sizeof(measurement64)), unsafe.Pointer(&measurement64))
	featVersions["VRAM_USAGE"] = uint64(measurement64)

	C.amdgpu_query_info(devHandle, C.AMDGPU_INFO_VRAM_GTT, C.uint(unsafe.Sizeof(info)), unsafe.Pointer(&info))
	featVersions["VRAM_SIZE"] = uint64(info.vram_size)
	featVersions["GTT_SIZE"] = uint64(info.gtt_size)

	return featVersions, nil
}

// GetAMDGPUs return a map of AMD GPU on a node identified by the part of the pci address
func GetAMDGPUs() []string {
	var cards []string

	if _, err := os.Stat("/sys/module/amdgpu/drivers/"); err != nil {
		return cards
	}

	//ex: /sys/module/amdgpu/drivers/pci:amdgpu/0000:19:00.0
	matches, _ := filepath.Glob("/sys/module/amdgpu/drivers/pci:amdgpu/[0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]:*")

	for _, path := range matches {
		devPaths, _ := filepath.Glob(path + "/drm/*")
		for _, devPath := range devPaths {
			switch name := filepath.Base(devPath); {
			case name[0:4] == "card":
				cards = append(cards, name)
				//devices[filepath.Base(path)][name[0:4]], _ = strconv.Atoi(name[4:])
			}
		}
	}
	return cards
}

func AMDGPU(cardName string) bool {
	sysfsVendorPath := "/sys/class/drm/" + cardName + "/device/vendor"
	b, err := os.ReadFile(sysfsVendorPath)
	if err == nil {
		vid := strings.TrimSpace(string(b))
		if "0x1002" == vid {
			return true
		}
	}
	return false
}

func openAMDGPU(cardName string) (C.amdgpu_device_handle, error) {
	if !AMDGPU(cardName) {
		return nil, fmt.Errorf("the %s is not an AMD GPU", cardName)
	}
	devPath := "/dev/dri/" + cardName

	dev, err := os.Open(devPath)

	if err != nil {
		return nil, fmt.Errorf("fail to open %s: %s", devPath, err)
	}
	defer dev.Close()

	devFd := C.int(dev.Fd())

	var devHandle C.amdgpu_device_handle
	var major C.uint32_t
	var minor C.uint32_t

	rc := C.amdgpu_device_initialize(devFd, &major, &minor, &devHandle)

	if rc < 0 {
		return nil, fmt.Errorf("fail to initialize %s: %d", devPath, err)
	}
	return devHandle, nil

}
