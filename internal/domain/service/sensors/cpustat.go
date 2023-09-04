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
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Info("Stopping RunPercentage")
			return ctx.Err()
		}
		err := g.getPercentageStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getPercentageStat(ctx context.Context, e *theme.Mesurement) error {
	var value float64 = 0

	select {
	case <-ctx.Done():
		g.log.Info("Stopping getPercentageStat")
		return ctx.Err()
	default:
		percent, err := cpu.PercentWithContext(ctx, e.Interval, false)
		if err != nil {
			return err
		}

		if len(percent) == 1 {
			value = percent[0]

			if e.Percent != nil && e.Percent.Show {
				img, x, y := BuildText(g.builder, value, "%3.0f", "%", e.Percent)
				g.jobs <- g.p.SendPayload(img, x, y)
			}
			if e.Text != nil && e.Text.Show {
				img, x, y := BuildText(g.builder, value, "%3.0f", "%", e.Text)
				g.jobs <- g.p.SendPayload(img, x, y)
			}
			if e.Radial != nil && e.Radial.Show {
				img, x, y := BuildRadial(g.builder, value, e.Radial)
				g.jobs <- g.p.SendPayload(img, x, y)
			}
			if e.Graph != nil && e.Graph.Show {
				img, x, y := BuildGraph(g.builder, value, e.Graph)
				g.jobs <- g.p.SendPayload(img, x, y)
			}
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
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Info("Stopping Frequency")
			return ctx.Err()
		}
		err := g.getFrequencyStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getFrequencyStat(ctx context.Context, e *theme.Mesurement) error {
	select {
	case <-ctx.Done():
		g.log.Info("Stopping getPercentageStat")
		return ctx.Err()
	default:
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
			g.jobs <- g.p.SendPayload(img, x, y)
		}
		if e.Text != nil && e.Text.Show {
			img, x, y := BuildTextFloat(g.builder, speed, utils.Hertz, e.Text)
			g.jobs <- g.p.SendPayload(img, x, y)
		}
		if e.Radial != nil && e.Radial.Show {
			img, x, y := BuildRadial(g.builder, speed, e.Radial)
			g.jobs <- g.p.SendPayload(img, x, y)
		}
		if e.Graph != nil && e.Graph.Show {
			img, x, y := BuildGraph(g.builder, speed, e.Graph)
			g.jobs <- g.p.SendPayload(img, x, y)
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
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Info("Stopping GpuStat")
			return ctx.Err()
		}
		err := g.getTemperatureStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getTemperatureStat(ctx context.Context, e *theme.Mesurement) error {
	select {
	case <-ctx.Done():
		g.log.Info("Stopping getPercentageStat")
		return ctx.Err()
	default:
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
			img, x, y := BuildText(g.builder, percent, "%3.0f", "%", e.Percent)
			g.jobs <- g.p.SendPayload(img, x, y)
		}
		if e.Text != nil && e.Text.Show {
			img, x, y := BuildText(g.builder, temperature, "%3.0f", "%", e.Text)
			g.jobs <- g.p.SendPayload(img, x, y)
		}
		if e.Radial != nil && e.Radial.Show {
			img, x, y := BuildRadial(g.builder, temperature, e.Radial)
			g.jobs <- g.p.SendPayload(img, x, y)
		}
		if e.Graph != nil && e.Graph.Show {
			img, x, y := BuildGraph(g.builder, temperature, e.Graph)
			g.jobs <- g.p.SendPayload(img, x, y)
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
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Info("Stopping GpuStat")
			return ctx.Err()
		}
		err := g.getLoadStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getLoadStat(ctx context.Context, e *theme.Load) error {
	select {
	case <-ctx.Done():
		g.log.Info("Stopping getPercentageStat")
		return ctx.Err()
	default:
		lload, err := load.AvgWithContext(ctx)
		if err != nil {
			return err
		}

		if e.One.Text != nil && e.One.Text.Show {
			img, x, y := BuildText(g.builder, lload.Load1, "%3.0f", "%", e.One.Text)
			g.jobs <- g.p.SendPayload(img, x, y)
		}
		if e.Five.Text != nil && e.Five.Text.Show {
			img, x, y := BuildText(g.builder, lload.Load5, "%3.0f", "%", e.Five.Text)
			g.jobs <- g.p.SendPayload(img, x, y)
		}
		if e.Fifteen.Text != nil && e.Fifteen.Text.Show {
			img, x, y := BuildText(g.builder, lload.Load15, "%3.0f", "%", e.Fifteen.Text)
			g.jobs <- g.p.SendPayload(img, x, y)
		}
	}
	return nil
}
