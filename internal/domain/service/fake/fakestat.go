package fake

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
	"time"
)

type FakeStat struct {
	ctx  context.Context
	log  *logger.Logger
	jobs chan<- any
}

func NewFakeStat(ctx context.Context, l *logger.Logger, j chan<- any) *FakeStat {
	return &FakeStat{
		ctx:  ctx,
		log:  l,
		jobs: j,
	}
}

func (g *FakeStat) Run(e entity.GPU) error {
	ticker := time.NewTicker(e.Interval)
	for {
		select {
		case <-ticker.C:
		case <-g.ctx.Done():
			g.log.Infof("Stopping GpuStat job...")
			return context.Canceled
		}
		return g.getStats(e)
	}
}

func (g *FakeStat) getStats(e entity.GPU) error {
	if e.StatProgressBars != nil {

	}
	if e.StatRadialBars != nil {

	}
	if e.StatTexts != nil {

	}
	close(g.jobs)

	return nil
}
