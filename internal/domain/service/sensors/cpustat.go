package sensors

import (
	"context"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
)

type CpuStat struct {
	log               *logger.Logger
	queue             *sender.RegionQueue
	builder           *local.Builder
	p                 *command.UpdatePayload
	encoding          command.PixelEncoding
	temperatureSensor string
}

func NewCpuStat(l *logger.Logger, q *sender.RegionQueue, b *local.Builder, p *command.UpdatePayload, encoding command.PixelEncoding, temperatureSensor string) *CpuStat {
	return &CpuStat{
		log:               l.With("runner", "cpu_stats"),
		queue:             q,
		builder:           b,
		p:                 p,
		encoding:          encoding,
		temperatureSensor: temperatureSensor,
	}
}

func (g *CpuStat) RunPercentage(ctx context.Context, e *theme.Mesurement) error {
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	err := g.getPercentageStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping RunPercentage")
			return ctx.Err()
		case <-ticker.C:
		}
		err := g.getPercentageStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getPercentageStat(ctx context.Context, e *theme.Mesurement) error {
	var value float64 = 0
	var payloads []*command.UpdatePayload

	percent, err := cpu.PercentWithContext(ctx, e.Interval, false)
	if err != nil {
		return err
	}

	if len(percent) == 1 {
		value = percent[0]

		if e.Percent != nil && e.Percent.Show {
			img, x, y := BuildText(g.builder, value, "%3.0f", "%", e.Percent)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Text != nil && e.Text.Show {
			img, x, y := BuildText(g.builder, value, "%3.0f", "%", e.Text)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Radial != nil && e.Radial.Show {
			img, x, y := BuildRadial(g.builder, value, e.Radial)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Graph != nil && e.Graph.Show {
			img, x, y := BuildGraph(g.builder, value, e.Graph)
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
			g.log.Info("stopping getPercentageStat")
			return ctx.Err()
		default:
			g.queue.Enqueue(payload)
		}
	}

	return nil
}

func (g *CpuStat) RunFrequency(ctx context.Context, e *theme.Mesurement) error {
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	err := g.getFrequencyStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping Frequency")
			return ctx.Err()
		case <-ticker.C:
		}

		err := g.getFrequencyStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getFrequencyStat(ctx context.Context, e *theme.Mesurement) error {
	var payloads []*command.UpdatePayload

	info, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return err
	}
	s := len(info)

	var vcpu float64 = 0
	for _, stat := range info {
		vcpu += stat.Mhz.Current
	}
	speed := vcpu / float64(s)

	if e.Percent != nil && e.Percent.Show {
		img, x, y := BuildText(g.builder, speed, "%3.0f", "%", e.Percent)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}
	if e.Text != nil && e.Text.Show {
		img, x, y := BuildTextFloat(g.builder, speed, utils.Hertz, e.Text)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}
	if e.Radial != nil && e.Radial.Show {
		img, x, y := BuildRadial(g.builder, speed, e.Radial)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}
	if e.Graph != nil && e.Graph.Show {
		img, x, y := BuildGraph(g.builder, speed, e.Graph)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getFrequencyStat")
			return ctx.Err()
		default:
			g.queue.Enqueue(payload)
		}
	}

	return nil
}

func (g *CpuStat) RunTemperature(ctx context.Context, e *theme.Mesurement) error {
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	err := g.getTemperatureStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping GpuStat")
			return ctx.Err()
		case <-ticker.C:
		}
		err := g.getTemperatureStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getTemperatureStat(ctx context.Context, e *theme.Mesurement) error {
	var payloads []*command.UpdatePayload

	cpuPatterns := []string{"cpu", "tdie"}
	temperature, percent := findSensorTemperature(ctx, g.temperatureSensor, cpuPatterns, g.log)

	if e.Percent != nil && e.Percent.Show {
		img, x, y := BuildText(g.builder, percent, "%3.0f", "°C", e.Percent)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}
	if e.Text != nil && e.Text.Show {
		img, x, y := BuildText(g.builder, temperature, "%3.0f", "°C", e.Text)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}
	if e.Radial != nil && e.Radial.Show {
		img, x, y := BuildRadial(g.builder, temperature, e.Radial)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}
	if e.Graph != nil && e.Graph.Show {
		img, x, y := BuildGraph(g.builder, temperature, e.Graph)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getTemperatureStat")
			return ctx.Err()
		default:
			g.queue.Enqueue(payload)
		}
	}
	return nil
}

func (g *CpuStat) RunLoad(ctx context.Context, e *theme.Load) error {
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	err := g.getLoadStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping GpuStat")
			return ctx.Err()
		case <-ticker.C:
		}
		err := g.getLoadStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getLoadStat(ctx context.Context, e *theme.Load) error {
	var payloads []*command.UpdatePayload

	lload, err := load.AvgWithContext(ctx)
	if err != nil {
		return err
	}

	if e.One.Text != nil && e.One.Text.Show {
		img, x, y := BuildText(g.builder, lload.Load1, "%3.0f", "%", e.One.Text)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}
	if e.Five.Text != nil && e.Five.Text.Show {
		img, x, y := BuildText(g.builder, lload.Load5, "%3.0f", "%", e.Five.Text)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}
	if e.Fifteen.Text != nil && e.Fifteen.Text.Show {
		img, x, y := BuildText(g.builder, lload.Load15, "%3.0f", "%", e.Fifteen.Text)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getTemperatureStat")
			return ctx.Err()
		default:
			g.queue.Enqueue(payload)
		}
	}
	return nil
}
