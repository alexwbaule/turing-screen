package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application"
	"github.com/alexwbaule/turing-screen/internal/application/hwinfo"
	"github.com/alexwbaule/turing-screen/internal/application/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/service/initializer"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sensors"
	"github.com/alexwbaule/turing-screen/internal/resource/gpu"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
	"github.com/alexwbaule/turing-screen/internal/resource/weather"
	"golang.org/x/sync/errgroup"
)

func main() {
	app := application.NewApplication()
	jobs := make(chan command.Command, 2000)
	defer close(jobs)

	app.Run(func(ctx context.Context) error {
		// --- Dependency Instantiation ---
		app.Log.Infof("device display: %#v", app.Config.GetDeviceDisplay())

		devSerial, err := serial.NewSerial(app.Config.GetDevicePort(), app.Log)
		if err != nil {
			log.Fatalf("failed to create serial sender: %v", err)
		}

		statsTheme, err := theme.NewTheme(app.Config, app.Log)
		if err != nil {
			return err
		}

		hw := hwinfo.Detect(app.Log, app.Config.GetGPUSensorConfig().Provider)
		staticTexts := statsTheme.GetStaticTexts()
		for key, st := range staticTexts {
			st.Text = hw.ReplaceText(st.Text)
			staticTexts[key] = st
		}

		builder := renderer.NewBuilder(app.Log, app.Config.GetDeviceDisplay(), statsTheme.GetDisplay())

		builder.BuildBackgroundImage(statsTheme.GetStaticImages())
		builder.BuildBackgroundTexts(statsTheme.GetStaticTexts())
		background := device.NewImageProcess(builder.GetBackground())

		cmdDevice := command.NewDevice(app.Log)
		cmdMedia := command.NewMedia(app.Log)
		cmdBright := command.NewBrightness(app.Log)
		cmdPayload := command.NewPayload(app.Log, statsTheme.GetDisplay().Orientation)
		cmdStorage := command.NewStorage(app.Log)
		cmdUpdate := command.NewUpdatePayload(app.Log, statsTheme.GetDisplay().Orientation, app.Config.GetDeviceDisplay())
		cmdOption := command.NewOption(app.Log)

		// --- End of Dependency Instantiation ---

		// 1. Create and run the synchronous initializer service
		initService := initializer.New(devSerial, app.Log, app.Config, statsTheme, cmdDevice, cmdMedia, cmdOption, cmdStorage, cmdBright, cmdPayload)
		if err := initService.Run(background); err != nil {
			return fmt.Errorf("device initialization failed: %w", err)
		}

		// If video mode is active, configure cmdUpdate to use overlay
		if overlay := initService.Overlay(); overlay != nil {
			cmdUpdate.SetOverlay(overlay)
		}

		// 2. Start the asynchronous worker for sensor updates
		worker := sender.NewWorker(devSerial, background, cmdDevice, cmdMedia, cmdPayload, app.Log)
		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			app.Log.Info("starting async worker for sensor updates")
			return worker.Run(ctx, jobs)
		})

		// Goroutine to handle graceful shutdown
		g.Go(func() error {
			<-ctx.Done()
			app.Log.Info("shutdown signal received.")
			if app.Config.GetTurnOffOnExit() {
				app.Log.Info("turning off device.")
				_ = worker.OffChannel(cmdDevice.TurnOff())
			}
			app.Log.Infof("cleaning queue with %d entries", len(jobs))
			// Drain the jobs channel for a short period
			timeout := time.After(500 * time.Millisecond)
			for {
				select {
				case <-jobs:
				case <-timeout:
					app.Log.Info("draining finished.")
					_ = devSerial.Close()
					return nil
				}
			}
		})
		// 3. Video overlay refresh goroutine (if video mode is active)
		if overlay := initService.Overlay(); overlay != nil {
			g.Go(func() error {
				app.Log.Info("starting video overlay refresh goroutine")
				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						app.Log.Info("video overlay refresh goroutine stopped")
						return nil
					case <-ticker.C:
						refreshCmd := overlay.Refresh()
						select {
						case jobs <- refreshCmd:
						default:
							app.Log.Warn("video overlay refresh dropped: jobs channel full")
						}
					}
				}
			})
		}
		// 4. Start sensor goroutines (they will feed the 'jobs' channel)
		app.Log.Info("initialization complete. starting sensor monitoring.")

		stats := statsTheme.GetStats()
		cpu := sensors.NewCpuStat(app.Log, jobs, builder, cmdUpdate, app.Config.GetCPUSensorConfig().TemperatureSensor)
		mem := sensors.NewMemStat(app.Log, jobs, builder, cmdUpdate)
		dt := sensors.NewDateTimeStat(app.Log, jobs, builder, cmdUpdate)
		net := sensors.NewDNetStat(app.Log, jobs, builder, cmdUpdate, app.Config.GetNetworkConfig())
		dsk := sensors.NewDiskStat(app.Log, jobs, builder, cmdUpdate, app.Config.GetDiskSensorConfig().TemperatureSensor)
		gpu := sensors.NewGpuStat(app.Log, jobs, builder, cmdUpdate, gpu.NewGPUProvider(app.Config.GetGPUSensorConfig().Provider, app.Log))

		weatherConfig := app.Config.GetWeatherConfig() // Pega a nova config

		if weatherConfig.Enabled && stats.Weather != nil {
			weatherClient := weather.NewClient()
			weatherSensor := sensors.NewWeatherSensor(app.Log, jobs, builder, cmdUpdate, weatherClient, weatherConfig.City)
			g.Go(func() error {
				return weatherSensor.Run(ctx, stats.Weather, weatherConfig.Interval)
			})
		}

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
		if stats.CPU.Load != nil {
			g.Go(func() error {
				app.Log.Info("starting worker CPU Load")
				return cpu.RunLoad(ctx, stats.CPU.Load)
			})
		}
		if stats.CPU.Fan != nil {
			g.Go(func() error {
				app.Log.Info("starting worker CPU Fan")
				return cpu.RunFan(ctx, stats.CPU.Fan)
			})
		}
		if stats.CPU.Power != nil {
			g.Go(func() error {
				app.Log.Info("starting worker CPU Power")
				return cpu.RunPower(ctx, stats.CPU.Power)
			})
		}
		if stats.CPU.Voltage != nil {
			g.Go(func() error {
				app.Log.Info("starting worker CPU Voltage")
				return cpu.RunVoltage(ctx, stats.CPU.Voltage)
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
		if stats.Volume != nil {
			vol := sensors.NewVolumeStat(app.Log, jobs, builder, cmdUpdate)
			g.Go(func() error {
				app.Log.Info("starting worker Volume")
				return vol.RunVolume(ctx, stats.Volume)
			})
		}

		return g.Wait()
	})
}
