package sensors

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	amdgpu "github.com/alexwbaule/turing-screen/internal/resource/gpu"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"time"
)

type GpuStat struct {
	log     *logger.Logger
	jobs    chan<- command.Command
	builder *local.Builder
	p       *command.UpdatePayload
}

func NewGpuStat(l *logger.Logger, j chan<- command.Command, b *local.Builder, p *command.UpdatePayload) *GpuStat {
	return &GpuStat{
		log:     l,
		jobs:    j,
		builder: b,
		p:       p,
	}
}

func (g *GpuStat) RunGpuStat(ctx context.Context, e *theme.GPU) error {
	g.log.Infof("Ticker: %s", e.Interval)
	ticker := time.NewTicker(e.Interval)

	err := g.getGpuStat(ctx, e)
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
		err := g.getGpuStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *GpuStat) getGpuStat(ctx context.Context, e *theme.GPU) error {

	v := amdgpu.GetAMDGPUs()

	familyName, err := amdgpu.GetCardFamilyName("card0")
	if err != nil {
		return err
	}
	g.log.Infof("CARD: %s", familyName)

	for s, m := range v {
		g.log.Infof("CARD: %s", s)
		for s2, i := range m {
			g.log.Infof("CARD: %s -> %d", s2, i)
		}
	}
	return nil
}
