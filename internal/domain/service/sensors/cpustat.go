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
	"github.com/shirou/gopsutil/v3/load"

	"time"
)

type CpuStat struct {
	log     *logger.Logger
	jobs    chan<- any
	builder *local.Builder
	p       *command.UpdatePayload
	u       *command.Media
}

func NewCpuStat(l *logger.Logger, j chan<- any, b *local.Builder, p *command.UpdatePayload, u *command.Media) *CpuStat {
	return &CpuStat{
		log:     l,
		jobs:    j,
		builder: b,
		p:       p,
		u:       u,
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
	if e.Text != nil && e.Text.Show {
		text := e.Text

		percent, err := cpu.PercentWithContext(ctx, e.Interval, false)
		if err != nil {
			return err
		}

		value := fmt.Sprintf("%3.0f", percent[0])
		if text.ShowUnit {
			value += "%"
		}

		img := g.builder.DrawText(value, text)
		imgUpdt := device.NewImageProcess(img)
		g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		g.jobs <- g.u.QueryStatus()
	}
	return nil
}

func (g *CpuStat) RunFrequency(ctx context.Context, e *theme.Mesurement) error {
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
		g.jobs <- g.u.QueryStatus()
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
		g.jobs <- g.u.QueryStatus()
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
		g.jobs <- g.u.QueryStatus()
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
		g.jobs <- g.u.QueryStatus()
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
		g.jobs <- g.u.QueryStatus()
	}
	return nil
}
