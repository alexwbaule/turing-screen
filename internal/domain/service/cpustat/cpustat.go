package cpustat

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
)

type CpuStat struct {
	ctx context.Context
	log *logger.Logger
}

func (g *CpuStat) Run(e map[string]entity.CPU) error {
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

func (g *CpuStat) getStats(e map[string]entity.CPU) error {
	return nil
}
