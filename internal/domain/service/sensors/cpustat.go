package sensors

import (
	"context"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
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
		log:     l,
		jobs:    j,
		builder: b,
		p:       p,
	}
}

func (g *CpuStat) RunPercentage(ctx context.Context, e *theme.Mesurement) error {
	g.log.Infof("Ticker: %s", e.Interval)
	ticker := time.NewTicker(e.Interval)

	err := g.getPercentageStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Infof("Stopping RunPercentage job...")
			return context.Canceled
		}
		err := g.getPercentageStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getPercentageStat(ctx context.Context, e *theme.Mesurement) error {
	percent, err := cpu.PercentWithContext(ctx, e.Interval, false)

	if len(percent) == 1 {
		prct := percent[0]

		if e.Text != nil && e.Text.Show {
			text := e.Text

			if err != nil {
				return err
			}

			value := fmt.Sprintf("%3.0f", prct)
			if text.ShowUnit {
				value += "%"
			}

			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
		if e.Graph != nil && e.Graph.Show {
			text := e.Graph
			img := g.builder.DrawProgressBar(prct, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
	}
	return nil
}

func (g *CpuStat) RunFrequency(ctx context.Context, e *theme.Mesurement) error {
	fmt.Printf("[%#v]\n", e)
	g.log.Infof("Ticker: %s", e.Interval)
	ticker := time.NewTicker(e.Interval)

	err := g.getFrequencyStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Infof("Stopping GpuStat job...")
			return context.Canceled
		}
		err := g.getFrequencyStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getFrequencyStat(ctx context.Context, e *theme.Mesurement) error {
	if e.Text != nil && e.Text.Show {
		text := e.Text

		info, err := cpu.InfoWithContext(ctx)
		if err != nil {
			return err
		}

		value := fmt.Sprintf("%3.0f", info[0].Mhz)
		if text.ShowUnit {
			value += "%"
		}

		img := g.builder.DrawText(value, text)
		imgUpdt := device.NewImageProcess(img)
		g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
	}
	return nil
}

func (g *CpuStat) RunTemperature(ctx context.Context, e *theme.Mesurement) error {
	g.log.Infof("Ticker: %s", e.Interval)
	ticker := time.NewTicker(e.Interval)

	err := g.getTemperatureStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Infof("Stopping GpuStat job...")
			return context.Canceled
		}
		err := g.getTemperatureStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getTemperatureStat(ctx context.Context, e *theme.Mesurement) error {
	var value string
	var has = false

	if e.Text != nil && e.Text.Show {
		text := e.Text

		stats, err := host.SensorsTemperaturesWithContext(ctx)
		if err != nil {
			return err
		}
		for _, stat := range stats {
			if stat.SensorKey == "zenpower_tdie" {
				g.log.Infof("Temperature: %s = %.2f", stat.SensorKey, stat.Temperature)
				g.log.Infof("Temperature: %.2f = %.2f", stat.High, stat.Critical)
				value = fmt.Sprintf("%3.0f", stat.Temperature)
				if text.ShowUnit {
					value += "°C"
				}
				has = true
			}
		}
		if has {
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
	}
	return nil
}

func (g *CpuStat) RunLoad(ctx context.Context, e *theme.Load) error {
	g.log.Infof("Ticker: %s", e.Interval)
	ticker := time.NewTicker(e.Interval)

	err := g.getLoadStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Infof("Stopping GpuStat job...")
			return context.Canceled
		}
		err := g.getLoadStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getLoadStat(ctx context.Context, e *theme.Load) error {
	lload, err := load.AvgWithContext(ctx)
	if err != nil {
		return err
	}

	if e.One.Text != nil && e.One.Text.Show {
		text := e.One.Text
		value := fmt.Sprintf("%3.0f", lload.Load1)
		if text.ShowUnit {
			value += "%"
		}
		img := g.builder.DrawText(value, text)
		imgUpdt := device.NewImageProcess(img)
		g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
	}

	if e.Five.Text != nil && e.Five.Text.Show {
		text := e.Five.Text
		value := fmt.Sprintf("%3.0f", lload.Load5)
		if text.ShowUnit {
			value += "%"
		}
		img := g.builder.DrawText(value, text)
		imgUpdt := device.NewImageProcess(img)
		g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
	}

	if e.Fifteen.Text != nil && e.Fifteen.Text.Show {
		text := e.Fifteen.Text
		value := fmt.Sprintf("%3.0f", lload.Load15)
		if text.ShowUnit {
			value += "%"
		}
		img := g.builder.DrawText(value, text)
		imgUpdt := device.NewImageProcess(img)
		g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
	}
	return nil
}
