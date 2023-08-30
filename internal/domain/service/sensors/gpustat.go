package sensors

import (
	"context"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	amdgpu "github.com/alexwbaule/turing-screen/internal/resource/gpu"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"time"
)

type GpuStat struct {
	log     *logger.Logger
	jobs    chan<- command.Command
	builder *local.Builder
	p       *command.UpdatePayload
}

func NewGpuStat(l *logger.Logger, j chan<- command.Command, b *local.Builder, p *command.UpdatePayload) *GpuStat {
	return &GpuStat{
		log:     l,
		jobs:    j,
		builder: b,
		p:       p,
	}
}

func (g *GpuStat) RunGpuStat(ctx context.Context, e *theme.GPU) error {
	ticker := time.NewTicker(e.Interval)

	err := g.getGpuStat(ctx, e)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			//g.log.Infof("Stopping RunMem job...")
			return context.Canceled
		}
		err := g.getGpuStat(ctx, e)
		if err != nil {
			return err
		}
	}
}

func (g *GpuStat) getGpuStat(ctx context.Context, e *theme.GPU) error {
	//g.log.Debugf("GPU: [%#v]", e)

	select {
	case <-ctx.Done():
		//g.log.Infof("Stopping getGpuStat job...")
		return context.Canceled
	default:
		var sensorMeasurements map[string]uint64
		var err error

		//var gpuAvgPower uint64 = 0
		var gpuTemp uint64 = 0
		var gpuLoad uint64 = 0
		var vranUsage uint64 = 0
		var vramSize uint64 = 0

		cards := amdgpu.GetAMDGPUs()

		if len(cards) > 0 {
			sensorMeasurements, err = amdgpu.GetCardSensor(cards[0])
			if err != nil {
				return err
			}
		}

		//if measurement, exists := sensorMeasurements["GPU_AVG_POWER"]; exists {
		//	gpuAvgPower = measurement
		//}
		if measurement, exists := sensorMeasurements["GPU_TEMP"]; exists {
			gpuTemp = measurement
		}
		if measurement, exists := sensorMeasurements["GPU_LOAD"]; exists {
			gpuLoad = measurement
		}
		if measurement, exists := sensorMeasurements["VRAM_USAGE"]; exists {
			vranUsage = measurement
		}
		if measurement, exists := sensorMeasurements["VRAM_SIZE"]; exists {
			vramSize = measurement
		}

		if e.Memory != nil {
			perc := uint64(0)
			if vramSize > 0 && vranUsage > 0 {
				perc = (vranUsage / vramSize) * 100
			}
			if e.Memory.Percent != nil && e.Memory.Percent.Show {
				text := e.Memory.Percent
				value := fmt.Sprintf("%3d", perc)
				if text.ShowUnit {
					value += "%"
				}
				img := g.builder.DrawText(value, text)
				imgUpdt := device.NewImageProcess(img)
				g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			}
			if e.Memory.Text != nil && e.Memory.Text.Show {
				text := e.Memory.Text
				value := fmt.Sprintf("%s", utils.Bytes(vranUsage))
				img := g.builder.DrawText(value, text)
				imgUpdt := device.NewImageProcess(img)
				g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			}
			if e.Memory.Graph != nil && e.Memory.Graph.Show {
				text := e.Memory.Graph
				img := g.builder.DrawProgressBar(float64(perc), text)
				imgUpdt := device.NewImageProcess(img)
				g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			}
		}
		if e.Temperature != nil {
			if e.Temperature.Percent != nil && e.Temperature.Percent.Show {
				text := e.Temperature.Percent
				value := fmt.Sprintf("%3d", gpuTemp)
				if text.ShowUnit {
					value += "°C"
				}
				img := g.builder.DrawText(value, text)
				imgUpdt := device.NewImageProcess(img)
				g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			}
			if e.Temperature.Text != nil && e.Temperature.Text.Show {
				text := e.Temperature.Text
				value := fmt.Sprintf("%3d", gpuTemp)
				if text.ShowUnit {
					value += "°C"
				}
				img := g.builder.DrawText(value, text)
				imgUpdt := device.NewImageProcess(img)
				g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			}
			if e.Temperature.Graph != nil && e.Temperature.Graph.Show {
				text := e.Temperature.Graph
				img := g.builder.DrawProgressBar(float64(gpuTemp), text)
				imgUpdt := device.NewImageProcess(img)
				g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			}
		}

		if e.Percentage != nil {
			if e.Percentage.Percent != nil && e.Percentage.Percent.Show {
				text := e.Percentage.Percent
				value := fmt.Sprintf("%3d", gpuLoad)
				if text.ShowUnit {
					value += "%"
				}
				img := g.builder.DrawText(value, text)
				imgUpdt := device.NewImageProcess(img)
				g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			}
			if e.Percentage.Text != nil && e.Percentage.Text.Show {
				text := e.Percentage.Text
				value := fmt.Sprintf("%3d", gpuLoad)
				if text.ShowUnit {
					value += "%"
				}
				img := g.builder.DrawText(value, text)
				imgUpdt := device.NewImageProcess(img)
				g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			}
			if e.Percentage.Graph != nil && e.Percentage.Graph.Show {
				text := e.Percentage.Graph
				img := g.builder.DrawProgressBar(float64(gpuLoad), text)
				imgUpdt := device.NewImageProcess(img)
				g.jobs <- g.p.SendPayload(imgUpdt, text.X, text.Y)
			}
		}
	}
	return nil
}
