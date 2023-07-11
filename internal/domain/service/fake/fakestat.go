package fake

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
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

func (g *FakeStat) Run(e theme.GPU) error {
	return g.getStats(e)
}

func (g *FakeStat) getStats(e theme.GPU) error {
	return nil
}
