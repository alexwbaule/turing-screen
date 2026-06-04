package sensors

import (
	"context"
	"strings"
	"time"

	"github.com/alexwbaule/gopsutil/v3/host"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/resource/interfaces"
)

type GpuStat struct {
	log      *logger.Logger
	jobs     chan<- command.Command
	builder  *renderer.Builder
	p        *command.UpdatePayload
	provider interfaces.Provider
}

func NewGpuStat(l *logger.Logger, j chan<- command.Command, b *renderer.Builder, p *command.UpdatePayload, provider interfaces.Provider) *GpuStat {
	return &GpuStat{
		log:      l.With("runner", "gpu_stats"),
		jobs:     j,
		builder:  b,
		p:        p,
		provider: provider,
	}
}

func (g *GpuStat) RunGpuStat(ctx context.Context, e *theme.GPU) error {
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	err := g.getGpuStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping RunGpuStat")
			return ctx.Err()
		case <-ticker.C:

		}
		err := g.getGpuStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *GpuStat) getGpuStat(ctx context.Context, e *theme.GPU) error {
	var payloads []*command.UpdatePayload

	var gpuAvgPower uint64 = 0
	var gpuTemp uint64 = 0
	var gpuLoad uint64 = 0
	var gpuFrequency uint64 = 0
	var gpuVoltage uint64 = 0
	var vranUsage uint64 = 0
	var vramSize uint64 = 0

	metrics, err := g.provider.GetMetrics()
	if err != nil {
		g.log.Warnf("failed to get GPU metrics: %v", err)
	} else {
		gpuTemp = metrics.Temperature
		gpuLoad = metrics.Load
		gpuAvgPower = metrics.Power
		gpuFrequency = metrics.Frequency
		gpuVoltage = metrics.Voltage
		vranUsage = metrics.VRAMUsage
		vramSize = metrics.VRAMSize
	}

	if e.Memory != nil {
		perc := float64(0)
		if vramSize > 0 && vranUsage > 0 {
			perc = float64(vranUsage) / float64(vramSize) * 100
		}
		payloads = append(payloads, BuildMesurement(g.builder, perc, "%3.f", "%", SizePercent, e.Memory, g.p)...)
	}
	if e.Temperature != nil {
		payloads = append(payloads, BuildMesurement(g.builder, float64(gpuTemp), "%3.f", "°C", SizeTemp, e.Temperature, g.p)...)
	}
	if e.Percentage != nil {
		payloads = append(payloads, BuildMesurement(g.builder, float64(gpuLoad), "%3.f", "%", SizePercent, e.Percentage, g.p)...)
	}
	if e.Power != nil {
		payloads = append(payloads, BuildMesurement(g.builder, float64(gpuAvgPower), "%3.f", "W", SizePower, e.Power, g.p)...)
	}
	if e.Frequency != nil {
		payloads = append(payloads, BuildMesurementFloat(g.builder, float64(gpuFrequency), utils.Hertz, SizeHertz, e.Frequency, g.p)...)
	}
	if e.Voltage != nil {
		payloads = append(payloads, BuildMesurement(g.builder, float64(gpuVoltage), "%4.f", "mV", 6, e.Voltage, g.p)...)
	}
	if e.Fan != nil {
		// Read GPU fan from hwmon
		var gpuFan float64
		fans, err := host.SensorsFansWithContext(ctx)
		if err == nil {
			for _, f := range fans {
				if strings.Contains(f.SensorKey, "amdgpu") || strings.Contains(f.SensorKey, "nvidia") {
					gpuFan = f.Speed
					break
				}
			}
		}
		payloads = append(payloads, BuildMesurement(g.builder, gpuFan, "%.0f", " RPM", SizeDefault, e.Fan, g.p)...)
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getGpuStat")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}
