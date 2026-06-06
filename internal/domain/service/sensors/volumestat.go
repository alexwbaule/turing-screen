package sensors

import (
	"context"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/resource/volume"
)

type VolumeStat struct {
	log      *logger.Logger
	jobs     chan<- command.Command
	builder  *renderer.Builder
	p        *command.UpdatePayload
	client   *volume.Client
	interval time.Duration
	lastVol  int
}

func NewVolumeStat(l *logger.Logger, j chan<- command.Command, b *renderer.Builder, p *command.UpdatePayload) *VolumeStat {
	return &VolumeStat{
		log:      l.With("runner", "volume_stats"),
		jobs:     j,
		builder:  b,
		p:        p,
		interval: 500 * time.Millisecond,
		lastVol:  -1, // force first update
	}
}

func (g *VolumeStat) RunVolume(ctx context.Context, e *theme.Volume) error {
	// Connect to PulseAudio/PipeWire
	client, err := volume.NewClient()
	if err != nil {
		g.log.Warnf("volume: failed to connect to PulseAudio: %v", err)
		return nil // Non-fatal: volume just won't display
	}
	g.client = client
	defer g.client.Close()

	// Poll using configured interval, only send update if changed
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping VolumeStat")
			return ctx.Err()
		case <-ticker.C:
			g.getVolume(ctx, e)
		}
	}
}

func (g *VolumeStat) getVolume(ctx context.Context, e *theme.Volume) {
	vol, err := g.client.GetVolume()
	if err != nil {
		g.log.Warnf("volume: %v", err)
		return
	}

	// Only send update if volume changed
	if vol == g.lastVol {
		return
	}
	g.lastVol = vol

	if e.Text != nil && e.Text.Show {
		if muted, _ := g.client.GetMuted(); muted {
			img, x, y := buildText(g.builder, "MUTE", e.Text, SizePercent)
			g.jobs <- g.p.SendPayload(img, x, y)
		} else {
			img, x, y := BuildText(g.builder, float64(vol), "%3.0f", "%", e.Text, SizePercent)
			select {
			case <-ctx.Done():
				return
			default:
				g.jobs <- g.p.SendPayload(img, x, y)
			}
		}
	}
}
