package sensors

import (
	"context"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/shirou/gopsutil/v3/disk"
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
		log:     l,
		jobs:    j,
		builder: b,
		p:       p,
	}
}

func (g *DiskStat) RunDiskStat(ctx context.Context, e *theme.Disk) error {
	g.log.Infof("Ticker: %s", e.Interval)
	ticker := time.NewTicker(e.Interval)

	err := g.getDiskStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			g.log.Infof("Stopping RunMem job...")
			return context.Canceled
		}
		err := g.getDiskStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *DiskStat) getDiskStat(ctx context.Context, e *theme.Disk) error {
	disks, err := disk.UsageWithContext(ctx, "/")
	if err != nil {
		return err
	}
	g.log.Infof("Disks: [%#v]", disks)

	if e.Free != nil {
		if e.Free.Text != nil && e.Free.Text.Show {
			text := e.Free.Text
			value := fmt.Sprintf("%s", utils.Bytes(disks.Free))
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
		if e.Free.Percent != nil && e.Free.Percent.Show {
			text := e.Free.Percent
			value := fmt.Sprintf("%3.f%%", 100-disks.UsedPercent)
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
	}
	if e.Used != nil {
		if e.Used.Text != nil && e.Used.Text.Show {
			text := e.Used.Text
			value := fmt.Sprintf("%s", utils.Bytes(disks.Used))
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
		if e.Used.Percent != nil && e.Used.Percent.Show {
			text := e.Used.Percent
			value := fmt.Sprintf("%3.f%%", disks.UsedPercent)
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
	}
	if e.Total != nil {
		if e.Total.Text != nil && e.Total.Text.Show {
			text := e.Total.Text
			value := fmt.Sprintf("%s", utils.Bytes(disks.Used))
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
		if e.Total.Percent != nil && e.Total.Percent.Show {
			text := e.Total.Percent
			value := fmt.Sprintf("%3d%%", 100)
			img := g.builder.DrawText(value, text)
			imgUpdt := device.NewImageProcess(img)
			g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
		}
	}
	return nil
}
