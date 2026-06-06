package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application"
	"github.com/alexwbaule/turing-screen/internal/application/hwinfo"
	appTheme "github.com/alexwbaule/turing-screen/internal/application/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/service/api"
	"github.com/alexwbaule/turing-screen/internal/domain/service/initializer"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sensors"
	gpuProvider "github.com/alexwbaule/turing-screen/internal/resource/gpu"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
	"github.com/alexwbaule/turing-screen/internal/resource/weather"
	"golang.org/x/sync/errgroup"
)

func main() {
	app := application.NewApplication()

	app.Run(func(ctx context.Context) error {
		// --- Dependency Instantiation (shared, always alive) ---
		app.Log.Infof("device display: %#v", app.Config.GetDeviceDisplay())

		devSerial, err := serial.NewSerial(app.Config.GetDevicePort(), app.Log)
		if err != nil {
			log.Fatalf("failed to create serial sender: %v", err)
		}

		cmdDevice := command.NewDevice(app.Log)
		cmdBright := command.NewBrightness(app.Log)
		cmdPayload := command.NewPayload(app.Log, 2) // LANDSCAPE default, overridden by theme

		// Shared jobs channel (single instance, reused across start/stop cycles)
		jobs := make(chan command.Command, 2000)

		// --- Sensor lifecycle management ---
		var sensorCancel context.CancelFunc
		var sensorWg sync.WaitGroup
		var sensorMu sync.Mutex
		sensorsRunning := false

		// startSensors initializes theme + sensors and runs them in background goroutines
		startSensors := func() {
			sensorMu.Lock()
			defer sensorMu.Unlock()
			if sensorsRunning {
				return
			}

			// Re-read config from disk (theme may have changed via API)
			if err := app.Config.Reload(); err != nil {
				app.Log.Warnf("failed to reload config: %v", err)
			}

			statsTheme, err := appTheme.NewTheme(app.Config, app.Log)
			if err != nil {
				app.Log.Errorf("failed to load theme: %v", err)
				return
			}

			hw := hwinfo.Detect(app.Log, app.Config.GetGPUSensorConfig().Provider)
			staticTexts := statsTheme.GetStaticTexts()
			for key, st := range staticTexts {
				st.Text = hw.ReplaceText(st.Text)
				staticTexts[key] = st
			}

			// Update payload orientation from theme
			orientation := statsTheme.GetDisplay().Orientation
			cmdPayload = command.NewPayload(app.Log, orientation)

			builder := renderer.NewBuilder(app.Log, app.Config.GetDeviceDisplay(), statsTheme.GetDisplay())
			builder.BuildBackgroundImage(statsTheme.GetStaticImages())
			builder.BuildBackgroundTexts(statsTheme.GetStaticTexts())
			background := device.NewImageProcess(builder.GetBackground())

			cmdMedia := command.NewMedia(app.Log)
			cmdStorage := command.NewStorage(app.Log)
			cmdOption := command.NewOption(app.Log)
			cmdUpdate := command.NewUpdatePayload(app.Log, orientation, app.Config.GetDeviceDisplay())

			// Initialize device
			initService := initializer.New(devSerial, app.Log, app.Config, statsTheme, cmdDevice, cmdMedia, cmdOption, cmdStorage, cmdBright, cmdPayload)
			if err := initService.Run(background); err != nil {
				app.Log.Errorf("device initialization failed: %v", err)
				return
			}

			if overlay := initService.Overlay(); overlay != nil {
				cmdUpdate.SetOverlay(overlay)
			}

			// Create sensor context (cancelable independently)
			var sensorCtx context.Context
			sensorCtx, sensorCancel = context.WithCancel(ctx)

			// Worker
			worker := sender.NewWorker(devSerial, background, cmdDevice, cmdMedia, cmdPayload, app.Log)
			sensorWg.Add(1)
			go func() {
				defer sensorWg.Done()
				if err := worker.Run(sensorCtx, jobs); err != nil && sensorCtx.Err() == nil {
					app.Log.Errorf("worker error: %v", err)
				}
			}()

			// Video overlay refresh
			if overlay := initService.Overlay(); overlay != nil {
				sensorWg.Add(1)
				go func() {
					defer sensorWg.Done()
					ticker := time.NewTicker(1 * time.Second)
					defer ticker.Stop()
					for {
						select {
						case <-sensorCtx.Done():
							return
						case <-ticker.C:
							select {
							case jobs <- overlay.Refresh():
							default:
							}
						}
					}
				}()
			}

			// Start sensor goroutines
			stats := statsTheme.GetStats()
			cpuCfg := app.Config.GetCPUSensorConfig()
			gpuCfg := app.Config.GetGPUSensorConfig()
			netCfg := app.Config.GetNetworkConfig()
			diskCfg := app.Config.GetDiskSensorConfig()
			memCfg := app.Config.GetMemoryConfig()

			cpuStat := sensors.NewCpuStat(app.Log, jobs, builder, cmdUpdate, cpuCfg.TemperatureSensor, cpuCfg.GetInterval())
			memStat := sensors.NewMemStat(app.Log, jobs, builder, cmdUpdate, memCfg.GetInterval())
			dtStat := sensors.NewDateTimeStat(app.Log, jobs, builder, cmdUpdate)
			netStat := sensors.NewDNetStat(app.Log, jobs, builder, cmdUpdate, netCfg, netCfg.GetInterval())
			dskStat := sensors.NewDiskStat(app.Log, jobs, builder, cmdUpdate, diskCfg.TemperatureSensor, diskCfg.GetInterval())
			gpuStat := sensors.NewGpuStat(app.Log, jobs, builder, cmdUpdate, gpuProvider.NewGPUProvider(gpuCfg.Provider, app.Log), gpuCfg.GetInterval())

			startSensor := func(name string, fn func(context.Context) error) {
				sensorWg.Add(1)
				go func() {
					defer sensorWg.Done()
					app.Log.Infof("starting worker %s", name)
					if err := fn(sensorCtx); err != nil && sensorCtx.Err() == nil {
						app.Log.Errorf("worker %s error: %v", name, err)
					}
				}()
			}

			if stats.CPU != nil {
				if stats.CPU.Percentage != nil {
					startSensor("CPU.Percentage", func(c context.Context) error { return cpuStat.RunPercentage(c, stats.CPU.Percentage) })
				}
				if stats.CPU.Frequency != nil {
					startSensor("CPU.Frequency", func(c context.Context) error { return cpuStat.RunFrequency(c, stats.CPU.Frequency) })
				}
				if stats.CPU.Temperature != nil {
					startSensor("CPU.Temperature", func(c context.Context) error { return cpuStat.RunTemperature(c, stats.CPU.Temperature) })
				}
				if stats.CPU.Load != nil {
					startSensor("CPU.Load", func(c context.Context) error { return cpuStat.RunLoad(c, stats.CPU.Load) })
				}
				if stats.CPU.Fan != nil {
					startSensor("CPU.Fan", func(c context.Context) error { return cpuStat.RunFan(c, stats.CPU.Fan) })
				}
				if stats.CPU.Power != nil {
					startSensor("CPU.Power", func(c context.Context) error { return cpuStat.RunPower(c, stats.CPU.Power) })
				}
				if stats.CPU.Voltage != nil {
					startSensor("CPU.Voltage", func(c context.Context) error { return cpuStat.RunVoltage(c, stats.CPU.Voltage) })
				}
			}
			if stats.GPU != nil {
				startSensor("GPU", func(c context.Context) error { return gpuStat.RunGpuStat(c, stats.GPU) })
			}
			if stats.Memory != nil {
				startSensor("Memory", func(c context.Context) error { return memStat.RunMemStat(c, stats.Memory) })
			}
			if stats.Date != nil {
				startSensor("DateTime", func(c context.Context) error { return dtStat.RunDateTime(c, stats.Date) })
			}
			if stats.Net != nil {
				startSensor("Network", func(c context.Context) error { return netStat.RunNetStat(c, stats.Net) })
			}
			if stats.Disk != nil {
				startSensor("Disk", func(c context.Context) error { return dskStat.RunDiskStat(c, stats.Disk) })
			}
			if stats.Volume != nil {
				volStat := sensors.NewVolumeStat(app.Log, jobs, builder, cmdUpdate)
				startSensor("Volume", func(c context.Context) error { return volStat.RunVolume(c, stats.Volume) })
			}

			weatherConfig := app.Config.GetWeatherConfig()
			if weatherConfig.Enabled && stats.Weather != nil {
				weatherClient := weather.NewClient()
				weatherSensor := sensors.NewWeatherSensor(app.Log, jobs, builder, cmdUpdate, weatherClient, weatherConfig.City)
				startSensor("Weather", func(c context.Context) error { return weatherSensor.Run(c, stats.Weather, weatherConfig.GetInterval()) })
			}

			sensorsRunning = true
			app.Log.Info("all sensors started")
		}

		// stopSensors cancels the sensor context and waits for goroutines to finish
		stopSensors := func() {
			sensorMu.Lock()
			defer sensorMu.Unlock()
			if !sensorsRunning {
				return
			}
			if sensorCancel != nil {
				sensorCancel()
			}
			sensorWg.Wait()
			// Drain remaining jobs
			for {
				select {
				case <-jobs:
				default:
					goto drained
				}
			}
		drained:
			sensorsRunning = false
			app.Log.Info("all sensors stopped, jobs drained")
		}

		// --- API Server (always runs) ---
		apiController := api.NewDaemonController(
			app.Log, devSerial, cmdDevice, cmdBright, cmdPayload,
			app.Config.GetThemeName(), "chs_5inch",
		)
		apiController.SetSensorControl(func() { stopSensors() }, func() { startSensors() })

		g, ctx := errgroup.WithContext(ctx)

		// API server goroutine
		apiServer := api.NewServer(app.Log, apiController, app.Config.GetAPIPort())
		g.Go(func() error {
			return apiServer.Start(ctx)
		})

		// Graceful shutdown handler
		g.Go(func() error {
			<-ctx.Done()
			app.Log.Info("shutdown signal received.")
			stopSensors()
			if app.Config.GetTurnOffOnExit() {
				app.Log.Info("turning off device.")
				devSerial.Execute(cmdDevice.TurnOff())
			}
			_ = devSerial.Close()
			return nil
		})

		// Start sensors on boot
		startSensors()

		return g.Wait()
	})
}
