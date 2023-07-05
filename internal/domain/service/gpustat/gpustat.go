package gpustat

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
	"time"
)

type GpuStat struct {
	ctx context.Context
}

func (g *GpuStat) Run(e entity.GPU) {
	ticker := time.NewTicker(e.Interval)
	for {
		select {
		case <-ticker.C:
		case <-g.ctx.Done():

		}
		g.getStats(e)
	}

}

func (g *GpuStat) getStats(e entity.GPU) {

}
