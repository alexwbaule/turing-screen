package sensors

import (
	"context"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/shirou/gopsutil/v3/mem"
	"time"
)

type MemStat struct {
	log     *logger.Logger
	jobs    chan<- any
	builder *local.Builder
	p       *command.UpdatePayload
	u       *command.Media
}

func NewMemStat(l *logger.Logger, j chan<- any, b *local.Builder, p *command.UpdatePayload, u *command.Media) *MemStat {
	return &MemStat{
		log:     l,
		jobs:    j,
		builder: b,
		p:       p,
		u:       u,
	}
}

func (g *MemStat) RunMem(ctx context.Context, e *theme.Memory) error {
	g.log.Infof("Ticker: %s", e.Interval)
	ticker := time.NewTicker(e.Interval)

	err := g.getMemStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Infof("Stopping RunMem job...")
			return context.Canceled
		}
		err := g.getMemStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *MemStat) getMemStat(ctx context.Context, e *theme.Memory) error {

	if e.Virtual != nil {
		virtualMem, err := mem.VirtualMemoryWithContext(ctx)
		if err != nil {
			return err
		}
		if e.Virtual.Free != nil && e.Virtual.Free.Show {
			text := e.Virtual.Free

			value := fmt.Sprintf("%5d", virtualMem.Available/1000000)
			if text.ShowUnit {
				value += "M"
			}

			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			g.jobs <- g.u.QueryStatus()
		}
		if e.Virtual.Used != nil && e.Virtual.Used.Show {
			text := e.Virtual.Used

			value := fmt.Sprintf("%5d", virtualMem.Used/1000000)
			if text.ShowUnit {
				value += "M"
			}

			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			g.jobs <- g.u.QueryStatus()
		}
		if e.Virtual.PercentText != nil && e.Virtual.PercentText.Show {
			text := e.Virtual.PercentText

			value := fmt.Sprintf("%3.0f", virtualMem.UsedPercent)
			if text.ShowUnit {
				value += "%"
			}

			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			g.jobs <- g.u.QueryStatus()
		}

	}
	if e.Swap != nil {

	}

	return nil
}
