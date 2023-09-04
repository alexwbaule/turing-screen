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

func main() {
	app := application.NewApplication()

	// Need a buffered, to be a 'non-blocking' channel
	// to not freezes on retry connect attempts.
	jobs := make(chan command.Command, 1500)
	defer close(jobs)

	app.Run(func(ctx context.Context) error {
		app.Log.Infof("Device: %#v", app.Config.GetDeviceDisplay())

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
			app.Log.Info("Starting Worker")
			return worker.Run(jobs, func() error {
				err := devSerial.RestartDevice()
				if err != nil {
					return err
				}
				go func() {
					jobs <- cmdDevice.Hello()
					app.Log.Info("Start Hello Again")
					jobs <- cmdMedia.StopVideo()
					app.Log.Info("Start StopVideo Again")
					jobs <- cmdMedia.StopMedia()
					app.Log.Info("Start StopMedia Again")
				}()
				return nil
			})
		})

		g.Go(func() error {
			<-ctx.Done()
			app.Log.Info("Shutdown device")
			write, err := devSerial.Write(cmdDevice.TurnOff())
			if err != nil {
				app.Log.Errorf("Can't send TurnOff Device, bytes [%d] -> %s", write, err)
			}
			err = devSerial.Close()
			if err != nil {
				app.Log.Errorf("Can't close serial port", err)
			}
			return nil
		})

		bg := builder.BuildBackgroundImage(statsTheme.GetStaticImages())
		fbg := builder.BuildBackgroundTexts(bg, statsTheme.GetStaticTexts())
		background := device2.NewImageProcess(fbg)

		app.Log.Info("Start App")

		jobs <- cmdDevice.Hello()
		app.Log.Info("Start Hello")

		jobs <- cmdMedia.StopVideo()
		app.Log.Info("Start StopVideo")

		jobs <- cmdMedia.StopMedia()
		app.Log.Info("Start StopMedia")

		jobs <- cmdBright.SetBrightness(app.Config.GetDeviceDisplay().Brightness)
		app.Log.Info("Start SetBrightness")

		jobs <- cmdPayload.SendPayload(background)
		app.Log.Info("Start SendPayload")

		stats := statsTheme.GetStats()

		cpu := sensors.NewCpuStat(app.Log, jobs, builder, cmdUpdate)
		mem := sensors.NewMemStat(app.Log, jobs, builder, cmdUpdate)
		dt := sensors.NewDateTimeStat(app.Log, jobs, builder, cmdUpdate)
		net := sensors.NewDNetStat(app.Log, jobs, builder, cmdUpdate, app.Config.GetNetworkConfig())
		dsk := sensors.NewDiskStat(app.Log, jobs, builder, cmdUpdate)
		gpu := sensors.NewGpuStat(app.Log, jobs, builder, cmdUpdate)

		if stats.CPU.Percentage != nil {
			g.Go(func() error {
				app.Log.Info("Starting Worker CPU Percentage")
				return cpu.RunPercentage(ctx, stats.CPU.Percentage)
			})
		}
		if stats.CPU.Frequency != nil {
			g.Go(func() error {
				app.Log.Info("Starting Worker CPU Frequency")
				return cpu.RunFrequency(ctx, stats.CPU.Frequency)
			})
		}
		if stats.CPU.Temperature != nil {
			g.Go(func() error {
				app.Log.Info("Starting Worker CPU Temperature")
				return cpu.RunTemperature(ctx, stats.CPU.Temperature)
			})
		}
		if stats.Memory != nil {
			g.Go(func() error {
				app.Log.Info("Starting Worker Memory")
				return mem.RunMemStat(ctx, stats.Memory)
			})
		}
		if stats.Date != nil {
			g.Go(func() error {
				app.Log.Info("Starting Worker Date")
				return dt.RunDateTime(ctx, stats.Date)
			})
		}
		if stats.Net != nil {
			g.Go(func() error {
				app.Log.Info("Starting Worker Net")
				return net.RunNetStat(ctx, stats.Net)
			})
		}
		if stats.Disk != nil {
			g.Go(func() error {
				app.Log.Info("Starting Worker Disk")
				return dsk.RunDiskStat(ctx, stats.Disk)
			})
		}
		if stats.GPU != nil {
			g.Go(func() error {
				app.Log.Info("Starting Worker GPU")
				return gpu.RunGpuStat(ctx, stats.GPU)
			})
		}
		return g.Wait()
	})
}
