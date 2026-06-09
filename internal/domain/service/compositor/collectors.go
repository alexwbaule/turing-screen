package compositor

import (
	"context"
	"strings"
	"time"

	"github.com/alexwbaule/gopsutil/v3/cpu"
	"github.com/alexwbaule/gopsutil/v3/disk"
	"github.com/alexwbaule/gopsutil/v3/host"
	"github.com/alexwbaule/gopsutil/v3/load"
	"github.com/alexwbaule/gopsutil/v3/mem"
	"github.com/alexwbaule/gopsutil/v3/net"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/resource/interfaces"
	"github.com/alexwbaule/turing-screen/internal/resource/weather"
)

// StartCollectors launches all sensor collector goroutines.
// Each collector updates values in the shared SensorValues struct.
func StartCollectors(
	ctx context.Context,
	log *logger.Logger,
	values *SensorValues,
	cpuTempSensor string,
	diskTempSensor string,
	netConfig device.Net,
	gpuProvider interfaces.Provider,
	weatherClient *weather.Client,
	weatherCity string,
	intervals struct {
		CPU, GPU, Memory, Disk, Network time.Duration
		Weather                         time.Duration
	},
) {
	// CPU collector
	go func() {
		ticker := time.NewTicker(intervals.CPU)
		defer ticker.Stop()
		collectCPU(ctx, values, cpuTempSensor, log)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collectCPU(ctx, values, cpuTempSensor, log)
			}
		}
	}()

	// GPU collector
	go func() {
		ticker := time.NewTicker(intervals.GPU)
		defer ticker.Stop()
		collectGPU(ctx, values, gpuProvider, log)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collectGPU(ctx, values, gpuProvider, log)
			}
		}
	}()

	// Memory collector
	go func() {
		ticker := time.NewTicker(intervals.Memory)
		defer ticker.Stop()
		collectMemory(ctx, values)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collectMemory(ctx, values)
			}
		}
	}()

	// Disk collector
	go func() {
		ticker := time.NewTicker(intervals.Disk)
		defer ticker.Stop()
		collectDisk(ctx, values, diskTempSensor, log)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collectDisk(ctx, values, diskTempSensor, log)
			}
		}
	}()

	// Network collector
	go func() {
		ticker := time.NewTicker(intervals.Network)
		defer ticker.Stop()
		var lastRecv, lastSent uint64
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				lastRecv, lastSent = collectNetwork(ctx, values, netConfig, intervals.Network, lastRecv, lastSent)
			}
		}
	}()

	// DateTime collector (500ms)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				values.mu.Lock()
				values.data.DateHour = now
				values.data.DateDay = now
				values.mu.Unlock()
			}
		}
	}()

	// Weather collector
	if weatherClient != nil {
		go func() {
			ticker := time.NewTicker(intervals.Weather)
			defer ticker.Stop()
			collectWeather(values, weatherClient, weatherCity, log)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					collectWeather(values, weatherClient, weatherCity, log)
				}
			}
		}()
	}
}

func collectCPU(ctx context.Context, values *SensorValues, tempSensor string, log *logger.Logger) {
	// Percentage
	percent, err := cpu.PercentWithContext(ctx, 0, false)
	if err == nil && len(percent) > 0 {
		values.mu.Lock()
		values.data.CPUPercent = percent[0]
		values.mu.Unlock()
	}

	// Frequency
	info, err := cpu.InfoWithContext(ctx)
	if err == nil && len(info) > 0 {
		var total float64
		for _, i := range info {
			total += i.Mhz.Current
		}
		values.mu.Lock()
		values.data.CPUFrequency = total / float64(len(info))
		values.mu.Unlock()
	}

	// Temperature
	temps, _ := host.SensorsTemperaturesWithContext(ctx)
	for _, t := range temps {
		if strings.Contains(t.SensorKey, "tdie") || strings.Contains(t.SensorKey, tempSensor) {
			values.mu.Lock()
			values.data.CPUTemp = t.Temperature
			values.mu.Unlock()
			break
		}
	}

	// Load
	lload, err := load.AvgWithContext(ctx)
	if err == nil {
		values.mu.Lock()
		values.data.CPULoad1 = lload.Load1
		values.data.CPULoad5 = lload.Load5
		values.data.CPULoad15 = lload.Load15
		values.mu.Unlock()
	}

	// Fan
	fans, _ := host.SensorsFansWithContext(ctx)
	for _, f := range fans {
		if f.Speed > 0 {
			values.mu.Lock()
			values.data.CPUFan = f.Speed
			values.mu.Unlock()
			break
		}
	}

	// Power
	powers, _ := host.SensorsPowerWithContext(ctx)
	for _, p := range powers {
		if strings.Contains(p.SensorKey, "core") || strings.Contains(p.SensorKey, "package") {
			values.mu.Lock()
			values.data.CPUPower = p.Power
			values.mu.Unlock()
			break
		}
	}

	// Voltage
	voltages, _ := host.SensorsVoltagesWithContext(ctx)
	for _, v := range voltages {
		if strings.Contains(v.SensorKey, "core") || strings.Contains(v.SensorKey, "vcore") {
			values.mu.Lock()
			values.data.CPUVoltage = v.Voltage
			values.mu.Unlock()
			break
		}
	}
}

func collectGPU(ctx context.Context, values *SensorValues, provider interfaces.Provider, log *logger.Logger) {
	if provider == nil {
		return
	}
	data, err := provider.GetMetrics()
	if err != nil || data == nil {
		return
	}

	values.mu.Lock()
	values.data.GPUPercent = float64(data.Load)
	values.data.GPUTemp = float64(data.Temperature)
	values.data.GPUPower = float64(data.Power)
	values.data.GPUFrequency = float64(data.Frequency)
	values.data.GPUVoltage = float64(data.Voltage)
	values.data.GPUFan = float64(data.Fan)
	if data.VRAMSize > 0 {
		values.data.GPUMemory = float64(data.VRAMUsage) / float64(data.VRAMSize) * 100
	}
	values.mu.Unlock()
}

func collectMemory(ctx context.Context, values *SensorValues) {
	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		values.mu.Lock()
		values.data.MemPercent = vmem.UsedPercent
		values.data.MemUsed = vmem.Used
		values.data.MemFree = vmem.Free
		values.mu.Unlock()
	}

	swap, err := mem.SwapMemoryWithContext(ctx)
	if err == nil {
		values.mu.Lock()
		values.data.SwapPercent = swap.UsedPercent
		values.mu.Unlock()
	}
}

func collectDisk(ctx context.Context, values *SensorValues, tempSensor string, log *logger.Logger) {
	usage, err := disk.UsageWithContext(ctx, "/")
	if err == nil {
		values.mu.Lock()
		values.data.DiskPercent = usage.UsedPercent
		values.data.DiskFree = 100 - usage.UsedPercent
		values.mu.Unlock()
	}

	temps, _ := host.SensorsTemperaturesWithContext(ctx)
	for _, t := range temps {
		if strings.Contains(t.SensorKey, "nvme") || strings.Contains(t.SensorKey, tempSensor) {
			values.mu.Lock()
			values.data.DiskTemp = t.Temperature
			values.mu.Unlock()
			break
		}
	}
}

func collectNetwork(ctx context.Context, values *SensorValues, netConfig device.Net, interval time.Duration, lastRecv, lastSent uint64) (uint64, uint64) {
	counters, err := net.IOCountersWithContext(ctx, true)
	if err != nil {
		return lastRecv, lastSent
	}

	for _, c := range counters {
		if c.Name == netConfig.Wired {
			recv := c.BytesRecv
			sent := c.BytesSent
			if lastRecv > 0 {
				values.mu.Lock()
				values.data.NetDownSpeed = float64(recv-lastRecv) / interval.Seconds()
				values.data.NetUpSpeed = float64(sent-lastSent) / interval.Seconds()
				values.data.NetDownloaded = recv
				values.data.NetUploaded = sent
				values.mu.Unlock()
			}
			return recv, sent
		}
	}
	return lastRecv, lastSent
}

func collectWeather(values *SensorValues, client *weather.Client, city string, log *logger.Logger) {
	forecast, err := client.GetCurrentWeather(city)
	if err != nil {
		log.Warnf("weather collect failed: %v", err)
		return
	}
	values.mu.Lock()
	values.data.WeatherTemp = forecast.Temperature
	values.data.WeatherDesc = forecast.Description
	values.data.WeatherWind = forecast.WindSpeed
	values.mu.Unlock()
}
