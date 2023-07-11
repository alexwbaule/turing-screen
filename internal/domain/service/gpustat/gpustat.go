package gpustat

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
)

type GpuStat struct {
	ctx context.Context
	log *logger.Logger
}

func (g *GpuStat) Run(e map[string]theme.GPU) error {
	/*
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
	*/
	return nil
}

func (g *GpuStat) getStats(e map[string]theme.GPU) error {
	return nil
}
