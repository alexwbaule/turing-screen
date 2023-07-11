package cpustat

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"time"
)

type CpuStat struct {
	ctx context.Context
	log *logger.Logger
}

func (g *CpuStat) Run(e theme.CPU) error {
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

func (g *CpuStat) getStats(e theme.CPU) error {
	return nil
}
