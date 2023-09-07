package sensors

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"time"
)

type CpuStat struct {
	log     *logger.Logger
	jobs    chan<- command.Command
	builder *local.Builder
	p       *command.UpdatePayload
}

func NewCpuStat(l *logger.Logger, j chan<- command.Command, b *local.Builder, p *command.UpdatePayload) *CpuStat {
	return &CpuStat{
		log:     l.With("runner", "cpu_stats"),
		jobs:    j,
		builder: b,
		p:       p,
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
			g.log.Info("Stopping RunPercentage")
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
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Text != nil && e.Text.Show {
			img, x, y := BuildText(g.builder, value, "%3.0f", "%", e.Text)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Radial != nil && e.Radial.Show {
			img, x, y := BuildRadial(g.builder, value, e.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Graph != nil && e.Graph.Show {
			img, x, y := BuildGraph(g.builder, value, e.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("Stopping getPercentageStat")
			return ctx.Err()
		default:
			g.jobs <- payload
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
			g.log.Info("Stopping Frequency")
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
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Text != nil && e.Text.Show {
		img, x, y := BuildTextFloat(g.builder, speed, utils.Hertz, e.Text)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Radial != nil && e.Radial.Show {
		img, x, y := BuildRadial(g.builder, speed, e.Radial)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Graph != nil && e.Graph.Show {
		img, x, y := BuildGraph(g.builder, speed, e.Graph)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("Stopping getFrequencyStat")
			return ctx.Err()
		default:
			g.jobs <- payload
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
			g.log.Info("Stopping GpuStat")
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

	var temperature float64 = 0
	var percent float64 = 0

	stats, err := host.SensorsTemperaturesWithContext(ctx)
	if err != nil {
		return err
	}
	for _, stat := range stats {
		if stat.SensorKey == "zenpower_tdie" {
			temperature = stat.Temperature
			percent = (stat.Temperature / stat.Critical) * 100
		}
	}

	if e.Percent != nil && e.Percent.Show {
		img, x, y := BuildText(g.builder, percent, "%3.0f", "°C", e.Percent)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Text != nil && e.Text.Show {
		img, x, y := BuildText(g.builder, temperature, "%3.0f", "°C", e.Text)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Radial != nil && e.Radial.Show {
		img, x, y := BuildRadial(g.builder, temperature, e.Radial)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Graph != nil && e.Graph.Show {
		img, x, y := BuildGraph(g.builder, temperature, e.Graph)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("Stopping getTemperatureStat")
			return ctx.Err()
		default:
			g.jobs <- payload
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
			g.log.Info("Stopping GpuStat")
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
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Five.Text != nil && e.Five.Text.Show {
		img, x, y := BuildText(g.builder, lload.Load5, "%3.0f", "%", e.Five.Text)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Fifteen.Text != nil && e.Fifteen.Text.Show {
		img, x, y := BuildText(g.builder, lload.Load15, "%3.0f", "%", e.Fifteen.Text)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("Stopping getTemperatureStat")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}
