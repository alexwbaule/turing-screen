package sensors

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"time"
)

type DateTimeStat struct {
	log     *logger.Logger
	jobs    chan<- any
	builder *local.Builder
	p       *command.UpdatePayload
	u       *command.Media
}

func NewDateTimeStat(l *logger.Logger, j chan<- any, b *local.Builder, p *command.UpdatePayload, u *command.Media) *DateTimeStat {
	return &DateTimeStat{
		log:     l,
		jobs:    j,
		builder: b,
		p:       p,
		u:       u,
	}
}

func (g *DateTimeStat) RunDateTime(ctx context.Context, e *theme.DateTime) error {
	g.log.Infof("Ticker: %s", e.Interval)
	ticker := time.NewTicker(e.Interval)

	err := g.getDateTime(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Infof("Stopping RunDateTime job...")
			return context.Canceled
		}
		err := g.getDateTime(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *DateTimeStat) getDateTime(ctx context.Context, e *theme.DateTime) error {
	now := time.Now()

	select {
	case <-ctx.Done():
		g.log.Infof("Stopping getDateTime job...")
		return context.Canceled
	default:
		if e.Day != nil {
			text := e.Day.Text
			value := now.Format(text.Format.String(theme.DATE))
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			g.jobs <- g.u.QueryStatus()
		}
		if e.Hour != nil {
			text := e.Hour.Text
			value := now.Format(text.Format.String(theme.TIME))
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			g.jobs <- g.u.QueryStatus()
		}
	}
	return nil
}
