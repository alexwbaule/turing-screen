package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alexwbaule/gopsutil/v3/cpu"
	"github.com/alexwbaule/gopsutil/v3/disk"
	"github.com/alexwbaule/gopsutil/v3/host"
	"github.com/alexwbaule/gopsutil/v3/mem"
	gnet "github.com/alexwbaule/gopsutil/v3/net"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/resource/gpu/amd"
	"github.com/alexwbaule/turing-screen/internal/resource/volume"
	"github.com/alexwbaule/turing-screen/internal/resource/weather"
)

func main() {
	ctx := context.Background()

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║       TURING SCREEN - SENSORS        ║")
	fmt.Println("╚══════════════════════════════════════╝")

	// --- CPU ---
	fmt.Println("\n━━━ CPU ━━━")
	pct, _ := cpu.PercentWithContext(ctx, time.Second, false)
	if len(pct) > 0 {
		fmt.Printf("  Usage:       %.1f%%\n", pct[0])
	}
	info, _ := cpu.InfoWithContext(ctx)
	if len(info) > 0 {
		var total float64
		for _, i := range info {
			total += i.Mhz.Current
		}
		fmt.Printf("  Frequency:   %s\n", utils.Hertz(total/float64(len(info)), true))
		fmt.Printf("  Model:       %s\n", info[0].ModelName)
	}
	temps, _ := host.SensorsTemperaturesWithContext(ctx)
	for _, t := range temps {
		if t.SensorKey == "zenpower_tdie" || t.SensorKey == "coretemp_package_id_0" {
			fmt.Printf("  Temperature: %.0f°C\n", t.Temperature)
			break
		}
	}
	powers, _ := host.SensorsPowerWithContext(ctx)
	for _, p := range powers {
		if p.SensorKey == "zenpower_svi2_p_core" || p.SensorKey == "rapl_package-0" {
			fmt.Printf("  Power:       %.1f W\n", p.Power)
			break
		}
	}
	voltages, _ := host.SensorsVoltagesWithContext(ctx)
	for _, v := range voltages {
		if v.SensorKey == "zenpower_svi2_core" {
			fmt.Printf("  Voltage:     %.3f V\n", v.Voltage)
			break
		}
	}
	fans, _ := host.SensorsFansWithContext(ctx)
	for _, f := range fans {
		if f.Speed > 0 {
			fmt.Printf("  Fan:         %.0f RPM (%s)\n", f.Speed, f.SensorKey)
			break
		}
	}

	// --- GPU ---
	fmt.Println("\n━━━ GPU ━━━")
	cards := amd.GetAMDGPUs()
	if len(cards) > 0 {
		sensors, err := amd.GetCardSensor(cards[0])
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		} else {
			fmt.Printf("  Temperature: %d°C\n", sensors["GPU_TEMP"])
			fmt.Printf("  Load:        %d%%\n", sensors["GPU_LOAD"])
			fmt.Printf("  Power:       %d W\n", sensors["GPU_AVG_POWER"])
			fmt.Printf("  Frequency:   %d MHz\n", sensors["GFX_SCLK"])
			fmt.Printf("  Voltage:     %d mV\n", sensors["VDDGFX"])
			fmt.Printf("  VRAM Used:   %.1f MB\n", float64(sensors["VRAM_USAGE"])/1024/1024)
			fmt.Printf("  VRAM Total:  %.1f MB\n", float64(sensors["VRAM_SIZE"])/1024/1024)
			if sensors["VRAM_SIZE"] > 0 {
				fmt.Printf("  VRAM %%:      %.1f%%\n", float64(sensors["VRAM_USAGE"])/float64(sensors["VRAM_SIZE"])*100)
			}
		}
		// GPU Fan
		for _, f := range fans {
			if f.SensorKey == "amdgpu_fan1" {
				fmt.Printf("  Fan:         %.0f RPM\n", f.Speed)
				break
			}
		}
	} else {
		fmt.Println("  No AMD GPU found (need root?)")
	}

	// --- Memory ---
	fmt.Println("\n━━━ MEMORY ━━━")
	vmem, _ := mem.VirtualMemoryWithContext(ctx)
	if vmem != nil {
		fmt.Printf("  Used:        %s\n", utils.BitsShort(vmem.Used, true))
		fmt.Printf("  Total:       %s\n", utils.BitsShort(vmem.Total, true))
		fmt.Printf("  Percent:     %.1f%%\n", vmem.UsedPercent)
	}

	// --- Disk ---
	fmt.Println("\n━━━ DISK ━━━")
	dsk, _ := disk.UsageWithContext(ctx, "/")
	if dsk != nil {
		fmt.Printf("  Used:        %s / %s\n", utils.Bytes(dsk.Used, true), utils.Bytes(dsk.Total, true))
		fmt.Printf("  Percent:     %.1f%%\n", dsk.UsedPercent)
	}
	for _, t := range temps {
		if t.SensorKey == "nvme_composite" {
			fmt.Printf("  Temperature: %.0f°C\n", t.Temperature)
			break
		}
	}

	// --- Network ---
	fmt.Println("\n━━━ NETWORK ━━━")
	counters1, _ := gnet.IOCountersWithContext(ctx, true)
	time.Sleep(time.Second)
	counters2, _ := gnet.IOCountersWithContext(ctx, true)
	for i, c2 := range counters2 {
		if i < len(counters1) {
			c1 := counters1[i]
			dl := float64(c2.BytesRecv - c1.BytesRecv)
			ul := float64(c2.BytesSent - c1.BytesSent)
			if dl > 0 || ul > 0 {
				fmt.Printf("  [%s] DL: %s  UL: %s\n", c2.Name,
					utils.NetSpeed(dl, true), utils.NetSpeed(ul, true))
			}
		}
	}

	// --- Weather ---
	fmt.Println("\n━━━ WEATHER ━━━")
	wc := weather.NewClient()
	forecast, err := wc.GetCurrentWeather("Sao Paulo")
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		fmt.Printf("  Temperature: %.1f°C\n", forecast.Temperature)
		fmt.Printf("  Condition:   %s (%s)\n", forecast.Condition, forecast.Description)
		fmt.Printf("  Wind:        %.1f km/h\n", forecast.WindSpeed)
	}

	// --- Volume ---
	fmt.Println("\n━━━ VOLUME ━━━")
	vc, err := volume.NewClient()
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
	} else {
		defer vc.Close()
		vol, err := vc.GetVolume()
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		} else {
			muted, _ := vc.GetMuted()
			fmt.Printf("  Volume:      %d%%\n", vol)
			fmt.Printf("  Muted:       %v\n", muted)
		}
	}

	// --- Date/Time ---
	fmt.Println("\n━━━ DATE/TIME ━━━")
	fmt.Printf("  Time:        %s\n", time.Now().Format("15:04:05"))
	fmt.Printf("  Date:        %s\n", time.Now().Format("2006-01-02"))

	fmt.Println("\n✔ Done")
}
