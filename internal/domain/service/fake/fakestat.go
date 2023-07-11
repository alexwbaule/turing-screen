package fake

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"time"
)

type FakeStat struct {
	ctx     context.Context
	log     *logger.Logger
	jobs    chan<- any
	builder local.Builder
}

func NewFakeStat(ctx context.Context, l *logger.Logger, j chan<- any, b local.Builder) *FakeStat {
	return &FakeStat{
		ctx:     ctx,
		log:     l,
		jobs:    j,
		builder: b,
	}
}

func (g *FakeStat) Run(e theme.GPU) error {
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

func (g *FakeStat) getStats(e theme.GPU) error {
	/*
		if e.Temperature.Text != nil {
			V := e.Temperature.Text
			img := g.builder.DrawText(fbg, fmt.Sprintf("%3d°C", rand.Intn(200)), *V)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- cmdUpdate.SendPayload(imgUpdt, V.X, V.Y)
			g.jobs <- cmdMedia.QueryStatus()
		}
	*/
	return nil
}
