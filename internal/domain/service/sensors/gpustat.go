package sensors

import (
	"context"
	"time"

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
			perc = float64(vranUsage/vramSize) * 100
		}
		if e.Memory.Percent != nil && e.Memory.Percent.Show {
			img, x, y := BuildText(g.builder, perc, "%3.f", "%", e.Memory.Percent, SizePercent)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Memory.Text != nil && e.Memory.Text.Show {
			img, x, y := BuildTextUint(g.builder, vranUsage, utils.Bytes, e.Memory.Text, SizeBytes)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Memory.Radial != nil && e.Memory.Radial.Show {
			img, x, y := BuildRadial(g.builder, perc, e.Memory.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Memory.Graph != nil && e.Memory.Graph.Show {
			img, x, y := BuildGraph(g.builder, perc, e.Memory.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}
	if e.Temperature != nil {
		if e.Temperature.Percent != nil && e.Temperature.Percent.Show {
			img, x, y := BuildText(g.builder, float64(gpuTemp), "%3.f", "%", e.Temperature.Percent, SizePercent)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Temperature.Text != nil && e.Temperature.Text.Show {
			img, x, y := BuildText(g.builder, float64(gpuTemp), "%3.f", "°C", e.Temperature.Text, SizeTemp)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Temperature.Radial != nil && e.Temperature.Radial.Show {
			img, x, y := BuildRadial(g.builder, float64(gpuTemp), e.Temperature.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Temperature.Graph != nil && e.Temperature.Graph.Show {
			img, x, y := BuildGraph(g.builder, float64(gpuTemp), e.Temperature.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}

	if e.Percentage != nil {
		if e.Percentage.Percent != nil && e.Percentage.Percent.Show {
			img, x, y := BuildText(g.builder, float64(gpuLoad), "%3.f", "%", e.Percentage.Percent, SizePercent)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Percentage.Text != nil && e.Percentage.Text.Show {
			img, x, y := BuildText(g.builder, float64(gpuLoad), "%3.f", "%", e.Percentage.Text, SizePercent)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Percentage.Radial != nil && e.Percentage.Radial.Show {
			img, x, y := BuildRadial(g.builder, float64(gpuLoad), e.Percentage.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Percentage.Graph != nil && e.Percentage.Graph.Show {
			img, x, y := BuildGraph(g.builder, float64(gpuLoad), e.Percentage.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}
	if e.Power != nil {
		if e.Power.Percent != nil && e.Power.Percent.Show {
			img, x, y := BuildText(g.builder, float64(gpuAvgPower), "%3.f", "%", e.Power.Percent, SizePercent)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Power.Text != nil && e.Power.Text.Show {
			img, x, y := BuildText(g.builder, float64(gpuAvgPower), "%3.f", "W", e.Power.Text, SizePower)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Power.Radial != nil && e.Power.Radial.Show {
			img, x, y := BuildRadial(g.builder, float64(gpuAvgPower), e.Power.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Power.Graph != nil && e.Power.Graph.Show {
			img, x, y := BuildGraph(g.builder, float64(gpuAvgPower), e.Power.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}
	if e.Frequency != nil {
		if e.Frequency.Text != nil && e.Frequency.Text.Show {
			img, x, y := BuildTextFloat(g.builder, float64(gpuFrequency), utils.Hertz, e.Frequency.Text, SizeHertz)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Frequency.Radial != nil && e.Frequency.Radial.Show {
			img, x, y := BuildRadial(g.builder, float64(gpuFrequency), e.Frequency.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Frequency.Graph != nil && e.Frequency.Graph.Show {
			img, x, y := BuildGraph(g.builder, float64(gpuFrequency), e.Frequency.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}
	if e.Voltage != nil {
		if e.Voltage.Text != nil && e.Voltage.Text.Show {
			img, x, y := BuildText(g.builder, float64(gpuVoltage), "%4.f", "mV", e.Voltage.Text, 6)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Voltage.Radial != nil && e.Voltage.Radial.Show {
			img, x, y := BuildRadial(g.builder, float64(gpuVoltage), e.Voltage.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Voltage.Graph != nil && e.Voltage.Graph.Show {
			img, x, y := BuildGraph(g.builder, float64(gpuVoltage), e.Voltage.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
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
