package gpustat

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
	"time"
)

type GpuStat struct {
	ctx context.Context
	log *logger.Logger
}

func (g *GpuStat) Run(e entity.GPU) error {
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

func (g *GpuStat) getStats(e entity.GPU) error {
	return nil
}
