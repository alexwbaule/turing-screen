package main

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application"
	"github.com/alexwbaule/turing-screen/internal/application/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sensors"
	device2 "github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
	"golang.org/x/sync/errgroup"
)

func init_device(jobs chan<- command.Command, cmddev *command.Device, cmdmed *command.Media) {
	jobs <- cmddev.Hello()
	jobs <- cmdmed.StopVideo()
	jobs <- cmdmed.StopMedia()
}

func main() {

	app := application.NewApplication()

	jobs := make(chan command.Command)

	app.Run(func(ctx context.Context) error {

		app.Log.Infof("Device: %+v", app.Config.GetDeviceDisplay())

		devSerial, err := serial.NewSerial(app.Config.GetDevicePort(), app.Log)
		if err != nil {
			app.Log.Fatal(err.Error())
		}

		statsTheme, err := theme.NewTheme(app.Config, app.Log)
		if err != nil {
			return err
		}

		worker := sender.NewWorker(ctx, devSerial, app.Log)

		cmdDevice := command.NewDevice(app.Log)
		cmdMedia := command.NewMedia(app.Log)
		cmdBright := command.NewBrightness(app.Log)
		cmdPayload := command.NewPayload(app.Log, statsTheme.GetDisplay().Orientation)
		cmdUpdate := command.NewUpdatePayload(app.Log, statsTheme.GetDisplay().Orientation, app.Config.GetDeviceDisplay())

		builder := local.NewBuilder(app.Log, app.Config.GetDeviceDisplay(), statsTheme.GetDisplay())

		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			//TODO: refazer isso, para tratar erro corretamente.
			err = worker.Run(jobs, func() error {
				err := devSerial.RestartDevice()
				app.Log.Infof("Device Restart: %+v", err)
				if err != nil {
					return err
				}
				init_device(jobs, cmdDevice, cmdMedia)
				return nil
			})
			return devSerial.Close()
		})
		init_device(jobs, cmdDevice, cmdMedia)
		bg := builder.BuildBackgroundImage(statsTheme.GetStaticImages())
		fbg := builder.BuildBackgroundTexts(bg, statsTheme.GetStaticTexts())
		background := device2.NewImageProcess(fbg)

		jobs <- cmdBright.SetBrightness(app.Config.GetDeviceDisplay().Brightness)
		jobs <- cmdPayload.SendPayload(background)

		stats := statsTheme.GetStats()

		cpu := sensors.NewCpuStat(app.Log, jobs, builder, cmdUpdate)
		mem := sensors.NewMemStat(app.Log, jobs, builder, cmdUpdate)
		dt := sensors.NewDateTimeStat(app.Log, jobs, builder, cmdUpdate)
		net := sensors.NewDNetStat(app.Log, jobs, builder, cmdUpdate, app.Config.GetNetworkConfig())
		dsk := sensors.NewDiskStat(app.Log, jobs, builder, cmdUpdate)
		gpu := sensors.NewGpuStat(app.Log, jobs, builder, cmdUpdate)

		if stats.CPU.Percentage != nil {
			g.Go(func() error {
				return cpu.RunPercentage(ctx, stats.CPU.Percentage)
			})
		}

		if stats.CPU.Frequency != nil {
			g.Go(func() error {
				return cpu.RunFrequency(ctx, stats.CPU.Frequency)
			})
		}
		if stats.CPU.Temperature != nil {
			g.Go(func() error {
				return cpu.RunTemperature(ctx, stats.CPU.Temperature)
			})
		}
		if stats.Memory != nil {
			g.Go(func() error {
				return mem.RunMemStat(ctx, stats.Memory)
			})
		}
		if stats.Date != nil {
			g.Go(func() error {
				return dt.RunDateTime(ctx, stats.Date)
			})
		}

		if stats.Net != nil {
			g.Go(func() error {
				return net.RunNetStat(ctx, stats.Net)
			})
		}

		if stats.Disk != nil {
			g.Go(func() error {
				return dsk.RunDiskStat(ctx, stats.Disk)
			})
		}
		if stats.GPU != nil {
			g.Go(func() error {
				return gpu.RunGpuStat(ctx, stats.GPU)
			})
		}

		return g.Wait()
	})
}
