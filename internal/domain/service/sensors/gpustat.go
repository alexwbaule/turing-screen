package sensors

import (
	"context"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	amdgpu "github.com/alexwbaule/turing-screen/internal/resource/gpu"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
)

type GpuStat struct {
	log      *logger.Logger
	queue    *sender.RegionQueue
	builder  *local.Builder
	p        *command.UpdatePayload
	encoding command.PixelEncoding
	provider amdgpu.GPUProvider
}

func NewGpuStat(l *logger.Logger, q *sender.RegionQueue, b *local.Builder, p *command.UpdatePayload, encoding command.PixelEncoding, provider amdgpu.GPUProvider) *GpuStat {
	return &GpuStat{
		log:      l.With("runner", "gpu_stats"),
		queue:    q,
		builder:  b,
		p:        p,
		encoding: encoding,
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
	var vranUsage uint64 = 0
	var vramSize uint64 = 0

	metrics, err := g.provider.GetMetrics()
	if err != nil {
		g.log.Warnf("failed to get GPU metrics: %v", err)
	} else {
		gpuTemp = metrics.Temperature
		gpuLoad = metrics.Load
		gpuAvgPower = metrics.Power
		vranUsage = metrics.VRAMUsage
		vramSize = metrics.VRAMSize
	}

	if e.Memory != nil {
		perc := float64(0)
		if vramSize > 0 && vranUsage > 0 {
			perc = float64(vranUsage/vramSize) * 100
		}
		if e.Memory.Percent != nil && e.Memory.Percent.Show {
			img, x, y := BuildText(g.builder, perc, "%3.f", "%", e.Memory.Percent)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Memory.Text != nil && e.Memory.Text.Show {
			img, x, y := BuildTextUint(g.builder, vranUsage, utils.Bytes, e.Memory.Text)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Memory.Radial != nil && e.Memory.Radial.Show {
			img, x, y := BuildRadial(g.builder, perc, e.Memory.Radial)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Memory.Graph != nil && e.Memory.Graph.Show {
			img, x, y := BuildGraph(g.builder, perc, e.Memory.Graph)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
	}
	if e.Temperature != nil {
		if e.Temperature.Percent != nil && e.Temperature.Percent.Show {
			img, x, y := BuildText(g.builder, float64(gpuTemp), "%3.f", "%", e.Temperature.Percent)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Temperature.Text != nil && e.Temperature.Text.Show {
			img, x, y := BuildText(g.builder, float64(gpuTemp), "%3.f", "°C", e.Temperature.Text)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Temperature.Radial != nil && e.Temperature.Radial.Show {
			img, x, y := BuildRadial(g.builder, float64(gpuTemp), e.Temperature.Radial)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Temperature.Graph != nil && e.Temperature.Graph.Show {
			img, x, y := BuildGraph(g.builder, float64(gpuTemp), e.Temperature.Graph)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
	}

	if e.Percentage != nil {
		if e.Percentage.Percent != nil && e.Percentage.Percent.Show {
			img, x, y := BuildText(g.builder, float64(gpuLoad), "%3.f", "%", e.Percentage.Percent)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Percentage.Text != nil && e.Percentage.Text.Show {
			img, x, y := BuildText(g.builder, float64(gpuLoad), "%3.f", "%", e.Percentage.Text)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Percentage.Radial != nil && e.Percentage.Radial.Show {
			img, x, y := BuildRadial(g.builder, float64(gpuLoad), e.Percentage.Radial)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Percentage.Graph != nil && e.Percentage.Graph.Show {
			img, x, y := BuildGraph(g.builder, float64(gpuLoad), e.Percentage.Graph)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
	}
	if e.Power != nil {
		if e.Power.Percent != nil && e.Power.Percent.Show {
			img, x, y := BuildText(g.builder, float64(gpuAvgPower), "%3.f", "%", e.Power.Percent)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Power.Text != nil && e.Power.Text.Show {
			img, x, y := BuildText(g.builder, float64(gpuAvgPower), "%3.f", "W", e.Power.Text)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Power.Radial != nil && e.Power.Radial.Show {
			img, x, y := BuildRadial(g.builder, float64(gpuAvgPower), e.Power.Radial)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Power.Graph != nil && e.Power.Graph.Show {
			img, x, y := BuildGraph(g.builder, float64(gpuAvgPower), e.Power.Graph)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getGpuStat")
			return ctx.Err()
		default:
			g.queue.Enqueue(payload)
		}
	}
	return nil
}
