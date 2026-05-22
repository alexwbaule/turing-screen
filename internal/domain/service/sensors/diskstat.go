package sensors

import (
	"context"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/shirou/gopsutil/v3/disk"
)

type DiskStat struct {
	log               *logger.Logger
	queue             *sender.RegionQueue
	builder           *local.Builder
	p                 *command.UpdatePayload
	encoding          command.PixelEncoding
	temperatureSensor string
}

func NewDiskStat(l *logger.Logger, q *sender.RegionQueue, b *local.Builder, p *command.UpdatePayload, encoding command.PixelEncoding, temperatureSensor string) *DiskStat {
	return &DiskStat{
		log:               l.With("runner", "disk_stats"),
		queue:             q,
		builder:           b,
		p:                 p,
		encoding:          encoding,
		temperatureSensor: temperatureSensor,
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
			g.log.Info("stopping RunDiskStat")
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
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Free.Text != nil && e.Free.Text.Show {
			img, x, y := BuildTextUint(g.builder, disks.Free, utils.Bytes, e.Free.Text)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Free.Radial != nil && e.Free.Radial.Show {
			img, x, y := BuildRadial(g.builder, 100-disks.UsedPercent, e.Free.Radial)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Free.Graph != nil && e.Free.Graph.Show {
			img, x, y := BuildGraph(g.builder, 100-disks.UsedPercent, e.Free.Graph)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
	}
	if e.Used != nil {
		if e.Used.Percent != nil && e.Used.Percent.Show {
			img, x, y := BuildText(g.builder, disks.UsedPercent, "%3.f", "%", e.Used.Percent)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Used.Text != nil && e.Used.Text.Show {
			img, x, y := BuildTextUint(g.builder, disks.Used, utils.Bytes, e.Used.Text)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Used.Radial != nil && e.Used.Radial.Show {
			img, x, y := BuildRadial(g.builder, disks.UsedPercent, e.Used.Radial)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Used.Graph != nil && e.Used.Graph.Show {
			img, x, y := BuildGraph(g.builder, disks.UsedPercent, e.Used.Graph)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
	}
	if e.Total != nil {
		if e.Total.Percent != nil && e.Total.Percent.Show {
			img, x, y := BuildText(g.builder, 100, "%3.f", "%", e.Total.Percent)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Total.Text != nil && e.Total.Text.Show {
			img, x, y := BuildTextUint(g.builder, disks.Total, utils.Bytes, e.Total.Text)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Total.Radial != nil && e.Total.Radial.Show {
			img, x, y := BuildRadial(g.builder, 100, e.Total.Radial)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Total.Graph != nil && e.Total.Graph.Show {
			img, x, y := BuildGraph(g.builder, 100, e.Total.Graph)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
	}
	if e.Temperature != nil {
		diskPatterns := []string{"nvme", "disk"}
		temperature, percent := findSensorTemperature(ctx, g.temperatureSensor, diskPatterns, g.log)

		if e.Temperature.Percent != nil && e.Temperature.Percent.Show {
			img, x, y := BuildText(g.builder, percent, "%3.0f", "%", e.Temperature.Percent)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Temperature.Text != nil && e.Temperature.Text.Show {
			img, x, y := BuildText(g.builder, temperature, "%3.0f", "°C", e.Temperature.Text)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Temperature.Radial != nil && e.Temperature.Radial.Show {
			img, x, y := BuildRadial(g.builder, temperature, e.Temperature.Radial)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
		if e.Temperature.Graph != nil && e.Temperature.Graph.Show {
			img, x, y := BuildGraph(g.builder, temperature, e.Temperature.Graph)
			p, err := g.p.SendPayload(img, x, y, g.encoding)
			if err != nil {
				return err
			}
			payloads = append(payloads, p)
		}
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getDiskStat")
			return ctx.Err()
		default:
			g.queue.Enqueue(payload)
		}
	}
	return nil
}
