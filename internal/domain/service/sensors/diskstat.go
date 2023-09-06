package sensors

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"time"
)

type DiskStat struct {
	log     *logger.Logger
	jobs    chan<- command.Command
	builder *local.Builder
	p       *command.UpdatePayload
}

func NewDiskStat(l *logger.Logger, j chan<- command.Command, b *local.Builder, p *command.UpdatePayload) *DiskStat {
	return &DiskStat{
		log:     l.With("runner", "disk_stats"),
		jobs:    j,
		builder: b,
		p:       p,
	}
}

func (g *DiskStat) RunDiskStat(ctx context.Context, e *theme.Disk) error {
	ticker := time.NewTicker(e.Interval)
	defer ticker.Stop()

	err := g.getDiskStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("Stopping RunDiskStat")
			return ctx.Err()
		case <-ticker.C:

		}
		err := g.getDiskStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *DiskStat) getDiskStat(ctx context.Context, e *theme.Disk) error {
	var payloads []*command.UpdatePayload

	disks, err := disk.UsageWithContext(ctx, "/")
	if err != nil {
		return err
	}
	if e.Free != nil {
		if e.Free.Percent != nil && e.Free.Percent.Show {
			img, x, y := BuildText(g.builder, 100-disks.UsedPercent, "%3.f", "%", e.Free.Percent)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Free.Text != nil && e.Free.Text.Show {
			img, x, y := BuildTextUint(g.builder, disks.Free, utils.Bytes, e.Free.Text)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Free.Radial != nil && e.Free.Radial.Show {
			img, x, y := BuildRadial(g.builder, 100-disks.UsedPercent, e.Free.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Free.Graph != nil && e.Free.Graph.Show {
			img, x, y := BuildGraph(g.builder, 100-disks.UsedPercent, e.Free.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}
	if e.Used != nil {
		if e.Used.Percent != nil && e.Used.Percent.Show {
			img, x, y := BuildText(g.builder, disks.UsedPercent, "%3.f", "%", e.Used.Percent)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Used.Text != nil && e.Used.Text.Show {
			img, x, y := BuildTextUint(g.builder, disks.Used, utils.Bytes, e.Used.Text)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Used.Radial != nil && e.Used.Radial.Show {
			img, x, y := BuildRadial(g.builder, disks.UsedPercent, e.Used.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Used.Graph != nil && e.Used.Graph.Show {
			img, x, y := BuildGraph(g.builder, disks.UsedPercent, e.Used.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}
	if e.Total != nil {
		if e.Total.Percent != nil && e.Total.Percent.Show {
			img, x, y := BuildText(g.builder, 100, "%3.f", "%", e.Total.Percent)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Total.Text != nil && e.Total.Text.Show {
			img, x, y := BuildTextUint(g.builder, disks.Total, utils.Bytes, e.Total.Text)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Total.Radial != nil && e.Total.Radial.Show {
			img, x, y := BuildRadial(g.builder, 100, e.Total.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Total.Graph != nil && e.Total.Graph.Show {
			img, x, y := BuildGraph(g.builder, 100, e.Total.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}
	if e.Temperature != nil {
		var temperature float64 = 0
		var percent float64 = 0

		stats, err := host.SensorsTemperaturesWithContext(ctx)
		if err != nil {
			return err
		}
		for _, stat := range stats {
			if stat.SensorKey == "nvme_composite" {
				temperature = stat.Temperature
				percent = (stat.Temperature / stat.Critical) * 100
			}
		}

		if e.Temperature.Percent != nil && e.Temperature.Percent.Show {
			img, x, y := BuildText(g.builder, percent, "%3.0f", "%", e.Temperature.Percent)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Temperature.Text != nil && e.Temperature.Text.Show {
			img, x, y := BuildText(g.builder, temperature, "%3.0f", "%", e.Temperature.Text)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Temperature.Radial != nil && e.Temperature.Radial.Show {
			img, x, y := BuildRadial(g.builder, temperature, e.Temperature.Radial)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
		if e.Temperature.Graph != nil && e.Temperature.Graph.Show {
			img, x, y := BuildGraph(g.builder, temperature, e.Temperature.Graph)
			payloads = append(payloads, g.p.SendPayload(img, x, y))
		}
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("Stopping getDiskStat")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}
