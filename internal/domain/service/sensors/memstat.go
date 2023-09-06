package sensors

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/shirou/gopsutil/v3/mem"
	"time"
)

type MemStat struct {
	log     *logger.Logger
	jobs    chan<- command.Command
	builder *local.Builder
	p       *command.UpdatePayload
}

func NewMemStat(l *logger.Logger, j chan<- command.Command, b *local.Builder, p *command.UpdatePayload) *MemStat {
	return &MemStat{
		log:     l.With("runner", "mem_stats"),
		jobs:    j,
		builder: b,
		p:       p,
	}
}

func (g *MemStat) RunMemStat(ctx context.Context, e *theme.Memory) error {
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	err := g.getMemStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("Stopping RunMem")
			return ctx.Err()
		case <-ticker.C:
		}
		err := g.getMemStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *MemStat) getMemStat(ctx context.Context, e *theme.Memory) error {
	var payloads []*command.UpdatePayload

	if e.Virtual != nil {
		virtualMem, err := mem.VirtualMemoryWithContext(ctx)
		if err != nil {
			return err
		}

		if e.Virtual.Free != nil && e.Virtual.Free.Show {
			img, x, y := BuildTextUint(g.builder, virtualMem.Available, utils.BitsShort, e.Virtual.Free)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Virtual.Used != nil && e.Virtual.Used.Show {
			img, x, y := BuildTextUint(g.builder, virtualMem.Used, utils.BitsShort, e.Virtual.Used)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Virtual.PercentText != nil && e.Virtual.PercentText.Show {
			img, x, y := BuildText(g.builder, virtualMem.UsedPercent, "%3.0f", "%", e.Virtual.PercentText)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Virtual.Graph != nil && e.Virtual.Graph.Show {
			img, x, y := BuildGraph(g.builder, virtualMem.UsedPercent, e.Virtual.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}
	if e.Swap != nil {
		swapMem, err := mem.SwapMemoryWithContext(ctx)
		if err != nil {
			return err
		}

		if e.Swap.Free != nil && e.Swap.Free.Show {
			img, x, y := BuildTextUint(g.builder, swapMem.Free, utils.BitsShort, e.Swap.Free)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Swap.Used != nil && e.Swap.Used.Show {
			img, x, y := BuildTextUint(g.builder, swapMem.Used, utils.BitsShort, e.Swap.Used)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Swap.PercentText != nil && e.Swap.PercentText.Show {
			img, x, y := BuildText(g.builder, swapMem.UsedPercent, "%3.0f", "%", e.Swap.PercentText)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Swap.Graph != nil && e.Swap.Graph.Show {
			img, x, y := BuildGraph(g.builder, swapMem.UsedPercent, e.Swap.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("Stopping getMemStat")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}
