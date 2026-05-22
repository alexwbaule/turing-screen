package sensors

import (
	"context"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
)

type DateTimeStat struct {
	log      *logger.Logger
	queue    *sender.RegionQueue
	builder  *local.Builder
	p        *command.UpdatePayload
	encoding command.PixelEncoding
}

func NewDateTimeStat(l *logger.Logger, q *sender.RegionQueue, b *local.Builder, p *command.UpdatePayload, encoding command.PixelEncoding) *DateTimeStat {
	return &DateTimeStat{
		log:      l.With("runner", "datetime_stats"),
		queue:    q,
		builder:  b,
		p:        p,
		encoding: encoding,
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
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}
	if e.Hour != nil {
		img, x, y := BuildTextDt(g.builder, t, theme.TIME, e.Hour.Text)
		p, err := g.p.SendPayload(img, x, y, g.encoding)
		if err != nil {
			return err
		}
		payloads = append(payloads, p)
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getDateTime")
			return ctx.Err()
		default:
			g.queue.Enqueue(payload)
		}
	}
	return nil
}
