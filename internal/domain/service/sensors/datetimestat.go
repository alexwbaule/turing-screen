package sensors

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
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
		log:     l.With("runner", "datetime_stats"),
		jobs:    j,
		builder: b,
		p:       p,
	}
}

func (g *DateTimeStat) RunDateTime(ctx context.Context, e *theme.DateTime) error {
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	err := g.getDateTime(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Info("stopping RunDateTime")
			return ctx.Err()
		}
		err := g.getDateTime(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *DateTimeStat) getDateTime(ctx context.Context, e *theme.DateTime) error {
	var payloads []*command.UpdatePayload
	t := time.Now()

	if e.Day != nil {
		img, x, y := BuildTextDt(g.builder, t, theme.DATE, e.Day.Text)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Hour != nil {
		img, x, y := BuildTextDt(g.builder, t, theme.TIME, e.Hour.Text)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getDateTime")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}
