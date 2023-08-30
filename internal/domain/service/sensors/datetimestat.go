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
	jobs    chan<- command.Command
	builder *local.Builder
	p       *command.UpdatePayload
}

func NewDateTimeStat(l *logger.Logger, j chan<- command.Command, b *local.Builder, p *command.UpdatePayload) *DateTimeStat {
	return &DateTimeStat{
		log:     l,
		jobs:    j,
		builder: b,
		p:       p,
	}
}

func (g *DateTimeStat) RunDateTime(ctx context.Context, e *theme.DateTime) error {
	ticker := time.NewTicker(e.Interval)

	err := g.getDateTime(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			//g.log.Infof("Stopping RunDateTime job...")
			return context.Canceled
		}
		err := g.getDateTime(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *DateTimeStat) getDateTime(ctx context.Context, e *theme.DateTime) error {
	//g.log.Debugf("DateTime: [%#v]", e)

	select {
	case <-ctx.Done():
		//g.log.Infof("Stopping getDateTime job...")
		return context.Canceled
	default:
		now := time.Now()
		if e.Day != nil {
			text := e.Day.Text
			value := now.Format(text.Format.String(theme.DATE))
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
		if e.Hour != nil {
			text := e.Hour.Text
			value := now.Format(text.Format.String(theme.TIME))
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
	}
	return nil
}
