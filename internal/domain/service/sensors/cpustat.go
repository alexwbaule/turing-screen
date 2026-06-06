package sensors

import (
	"context"
	"strings"
	"time"

	"github.com/alexwbaule/gopsutil/v3/cpu"
	"github.com/alexwbaule/gopsutil/v3/host"
	"github.com/alexwbaule/gopsutil/v3/load"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/utils"
)

type CpuStat struct {
	log               *logger.Logger
	jobs              chan<- command.Command
	builder           *renderer.Builder
	p                 *command.UpdatePayload
	temperatureSensor string
	interval          time.Duration
}

func NewCpuStat(l *logger.Logger, j chan<- command.Command, b *renderer.Builder, p *command.UpdatePayload, temperatureSensor string, interval time.Duration) *CpuStat {
	return &CpuStat{
		log:               l.With("runner", "cpu_stats"),
		jobs:              j,
		builder:           b,
		p:                 p,
		temperatureSensor: temperatureSensor,
		interval:          interval,
	}
}

func (g *CpuStat) RunPercentage(ctx context.Context, e *theme.Mesurement) error {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	err := g.getPercentageStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping RunPercentage")
			return ctx.Err()
		case <-ticker.C:
		}
		err := g.getPercentageStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getPercentageStat(ctx context.Context, e *theme.Mesurement) error {
	var value float64 = 0
	var payloads []*command.UpdatePayload

	percent, err := cpu.PercentWithContext(ctx, g.interval, false)
	if err != nil {
		return err
	}

	if len(percent) == 1 {
		value = percent[0]
		payloads = BuildMesurement(g.builder, value, "%3.0f", "%", SizePercent, e, g.p, "CPU.Percentage")
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getPercentageStat")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}

	return nil
}

func (g *CpuStat) RunFrequency(ctx context.Context, e *theme.Mesurement) error {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	err := g.getFrequencyStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping Frequency")
			return ctx.Err()
		case <-ticker.C:
		}

		err := g.getFrequencyStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getFrequencyStat(ctx context.Context, e *theme.Mesurement) error {
	var payloads []*command.UpdatePayload

	info, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return err
	}
	s := len(info)

	var vcpu float64 = 0
	for _, stat := range info {
		vcpu += stat.Mhz.Current
	}
	speed := vcpu / float64(s)

	payloads = BuildMesurementFloat(g.builder, speed, utils.Hertz, SizeHertz, e, g.p, "CPU.Frequency")

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getFrequencyStat")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}

	return nil
}

func (g *CpuStat) RunTemperature(ctx context.Context, e *theme.Mesurement) error {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	err := g.getTemperatureStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping GpuStat")
			return ctx.Err()
		case <-ticker.C:
		}
		err := g.getTemperatureStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getTemperatureStat(ctx context.Context, e *theme.Mesurement) error {
	var payloads []*command.UpdatePayload

	cpuPatterns := []string{"cpu", "tdie", "zenpower_tdie"}
	temperature, _ := findSensorTemperature(ctx, g.temperatureSensor, cpuPatterns, g.log)

	payloads = BuildMesurement(g.builder, temperature, "%3.0f", "°C", SizeTemp, e, g.p, "CPU.Temperature")

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getTemperatureStat")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}

func (g *CpuStat) RunLoad(ctx context.Context, e *theme.Load) error {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	err := g.getLoadStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping GpuStat")
			return ctx.Err()
		case <-ticker.C:
		}
		err := g.getLoadStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *CpuStat) getLoadStat(ctx context.Context, e *theme.Load) error {
	var payloads []*command.UpdatePayload

	lload, err := load.AvgWithContext(ctx)
	if err != nil {
		return err
	}

	if e.One != nil && e.One.Text != nil && e.One.Text.Show {
		img, x, y := BuildText(g.builder, lload.Load1, "%3.0f", "%", e.One.Text, SizePercent)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Five != nil && e.Five.Text != nil && e.Five.Text.Show {
		img, x, y := BuildText(g.builder, lload.Load5, "%3.0f", "%", e.Five.Text, SizePercent)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}
	if e.Fifteen != nil && e.Fifteen.Text != nil && e.Fifteen.Text.Show {
		img, x, y := BuildText(g.builder, lload.Load15, "%3.0f", "%", e.Fifteen.Text, SizePercent)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			g.log.Info("stopping getTemperatureStat")
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}

func (g *CpuStat) RunFan(ctx context.Context, e *theme.Mesurement) error {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	if err := g.getFanStat(ctx, e); err != nil {
		g.log.Warnf("failed to get initial CPU fan: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping RunFan")
			return ctx.Err()
		case <-ticker.C:
		}
		if err := g.getFanStat(ctx, e); err != nil {
			g.log.Warnf("failed to get CPU fan: %v", err)
		}
	}
}

func (g *CpuStat) getFanStat(ctx context.Context, e *theme.Mesurement) error {
	var payloads []*command.UpdatePayload

	fans, err := host.SensorsFansWithContext(ctx)
	if err != nil {
		return err
	}

	// Find CPU fan: look for nct/it87/nuvoton (motherboard super I/O) fan1
	var fanSpeed float64
	for _, f := range fans {
		if strings.Contains(f.SensorKey, "nct") || strings.Contains(f.SensorKey, "it87") || strings.Contains(f.SensorKey, "nuvoton") {
			if f.Speed > 0 {
				fanSpeed = f.Speed
				break
			}
		}
	}

	if e.Text != nil && e.Text.Show {
		img, x, y := BuildText(g.builder, fanSpeed, "%.0f", " RPM", e.Text, SizeDefault)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}

func (g *CpuStat) RunPower(ctx context.Context, e *theme.Mesurement) error {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	if err := g.getPowerStat(ctx, e); err != nil {
		g.log.Warnf("failed to get initial CPU power: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping RunPower")
			return ctx.Err()
		case <-ticker.C:
		}
		if err := g.getPowerStat(ctx, e); err != nil {
			g.log.Warnf("failed to get CPU power: %v", err)
		}
	}
}

func (g *CpuStat) getPowerStat(ctx context.Context, e *theme.Mesurement) error {
	var payloads []*command.UpdatePayload

	powers, err := host.SensorsPowerWithContext(ctx)
	if err != nil {
		return err
	}

	// Find CPU package power (zenpower/rapl core)
	var watts float64
	for _, p := range powers {
		if strings.Contains(p.SensorKey, "core") || strings.Contains(p.SensorKey, "package") {
			watts = p.Power
			break
		}
	}

	if e.Text != nil && e.Text.Show {
		img, x, y := BuildText(g.builder, watts, "%.0f", "W", e.Text, SizeDefault)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}

func (g *CpuStat) RunVoltage(ctx context.Context, e *theme.Mesurement) error {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	if err := g.getVoltageStat(ctx, e); err != nil {
		g.log.Warnf("failed to get initial CPU voltage: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			g.log.Info("stopping RunVoltage")
			return ctx.Err()
		case <-ticker.C:
		}
		if err := g.getVoltageStat(ctx, e); err != nil {
			g.log.Warnf("failed to get CPU voltage: %v", err)
		}
	}
}

func (g *CpuStat) getVoltageStat(ctx context.Context, e *theme.Mesurement) error {
	var payloads []*command.UpdatePayload

	voltages, err := host.SensorsVoltagesWithContext(ctx)
	if err != nil {
		return err
	}

	// Find CPU core voltage (zenpower svi2_core or vcore)
	var volts float64
	for _, v := range voltages {
		if strings.Contains(v.SensorKey, "core") || strings.Contains(v.SensorKey, "vcore") {
			volts = v.Voltage
			break
		}
	}

	if e.Text != nil && e.Text.Show {
		img, x, y := BuildText(g.builder, volts, "%.3f", "V", e.Text, SizeDefault)
		payloads = append(payloads, g.p.SendPayload(img, x, y))
	}

	for _, payload := range payloads {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			g.jobs <- payload
		}
	}
	return nil
}
