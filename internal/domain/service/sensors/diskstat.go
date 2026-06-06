package sensors

import (
	"context"
	"time"

	"github.com/alexwbaule/gopsutil/v3/disk"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
)

type DiskStat struct {
	log               *logger.Logger
	jobs              chan<- command.Command
	builder           *renderer.Builder
	p                 *command.UpdatePayload
	temperatureSensor string
	interval          time.Duration
}

func NewDiskStat(l *logger.Logger, j chan<- command.Command, b *renderer.Builder, p *command.UpdatePayload, temperatureSensor string, interval time.Duration) *DiskStat {
	return &DiskStat{
		log:               l.With("runner", "disk_stats"),
		jobs:              j,
		builder:           b,
		p:                 p,
		temperatureSensor: temperatureSensor,
		interval:          interval,
	}
}

func (g *DiskStat) RunDiskStat(ctx context.Context, e *theme.Disk) error {
	ticker := time.NewTicker(g.interval)
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
		payloads = append(payloads, BuildMesurement(g.builder, 100-disks.UsedPercent, "%3.f", "%", SizePercent, e.Free, g.p, "Disk.Free")...)
	}
	if e.Used != nil {
		payloads = append(payloads, BuildMesurement(g.builder, disks.UsedPercent, "%3.f", "%", SizePercent, e.Used, g.p, "Disk.Used")...)
	}
	if e.Total != nil {
		payloads = append(payloads, BuildMesurement(g.builder, 100, "%3.f", "%", SizePercent, e.Total, g.p, "Disk.Total")...)
	}
	if e.Temperature != nil {
		diskPatterns := []string{"nvme", "disk", "nvme_composite"}
		temperature, _ := findSensorTemperature(ctx, g.temperatureSensor, diskPatterns, g.log)
		payloads = append(payloads, BuildMesurement(g.builder, temperature, "%3.0f", "°C", SizeTemp, e.Temperature, g.p, "Disk.Temperature")...)
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getDiskStat")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}
