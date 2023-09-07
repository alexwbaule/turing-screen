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
	"time"
)

func main() {
	app := application.NewApplication()

	// Need a buffered, to be a 'non-blocking' channel
	// to not freezes on retry connect attempts.
	jobs := make(chan command.Command)

	defer close(jobs)

	app.Run(func(ctx context.Context) error {
		app.Log.Infof("device display: %#v", app.Config.GetDeviceDisplay())

		devSerial, err := serial.NewSerial(app.Config.GetDevicePort(), app.Log)
		if err != nil {
			app.Log.Fatal(err.Error())
		}

		statsTheme, err := theme.NewTheme(app.Config, app.Log)
		if err != nil {
			return err
		}
		builder := local.NewBuilder(app.Log, app.Config.GetDeviceDisplay(), statsTheme.GetDisplay())

		bg := builder.BuildBackgroundImage(statsTheme.GetStaticImages())
		fbg := builder.BuildBackgroundTexts(bg, statsTheme.GetStaticTexts())
		background := device2.NewImageProcess(fbg)

		cmdDevice := command.NewDevice(app.Log)
		cmdMedia := command.NewMedia(app.Log)
		cmdBright := command.NewBrightness(app.Log)
		cmdPayload := command.NewPayload(app.Log, statsTheme.GetDisplay().Orientation)
		cmdUpdate := command.NewUpdatePayload(app.Log, statsTheme.GetDisplay().Orientation, app.Config.GetDeviceDisplay())
		worker := sender.NewWorker(ctx, devSerial, background, cmdDevice, cmdMedia, cmdPayload, app.Log)

		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			app.Log.Info("starting reader worker")
			return worker.Run(jobs)
		})

		g.Go(func() error {
			<-ctx.Done()
			app.Log.Info("shutdown device")
			_ = worker.OffChannel(cmdDevice.TurnOff())

			count := 0
			for {
				select {
				case _ = <-jobs:
				default:
					time.Sleep(200 * time.Millisecond)
					count++
				}
				if count == 8 {
					app.Log.Info("empty messages in queue")
					break
				}
			}
			_ = devSerial.Close()
			return ctx.Err()
		})

		app.Log.Info("starting app")
		jobs <- cmdDevice.Hello()
		jobs <- cmdMedia.StopVideo()
		jobs <- cmdMedia.StopMedia()
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
				app.Log.Info("starting worker CPU Percentage")
				return cpu.RunPercentage(ctx, stats.CPU.Percentage)
			})
		}
		if stats.CPU.Frequency != nil {
			g.Go(func() error {
				app.Log.Info("starting worker CPU Frequency")
				return cpu.RunFrequency(ctx, stats.CPU.Frequency)
			})
		}
		if stats.CPU.Temperature != nil {
			g.Go(func() error {
				app.Log.Info("starting worker CPU Temperature")
				return cpu.RunTemperature(ctx, stats.CPU.Temperature)
			})
		}
		if stats.Memory != nil {
			g.Go(func() error {
				app.Log.Info("starting worker Memory")
				return mem.RunMemStat(ctx, stats.Memory)
			})
		}
		if stats.Date != nil {
			g.Go(func() error {
				app.Log.Info("starting worker Date")
				return dt.RunDateTime(ctx, stats.Date)
			})
		}
		if stats.Net != nil {
			g.Go(func() error {
				app.Log.Info("starting worker Net")
				return net.RunNetStat(ctx, stats.Net)
			})
		}
		if stats.Disk != nil {
			g.Go(func() error {
				app.Log.Info("starting worker Disk")
				return dsk.RunDiskStat(ctx, stats.Disk)
			})
		}
		if stats.GPU != nil {
			g.Go(func() error {
				app.Log.Info("starting worker GPU")
				return gpu.RunGpuStat(ctx, stats.GPU)
			})
		}
		return g.Wait()
	})
}
