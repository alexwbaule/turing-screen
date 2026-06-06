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
	"github.com/alexwbaule/turing-screen/internal/domain/service/compositor"
	"github.com/alexwbaule/turing-screen/internal/domain/service/initializer"
	"github.com/alexwbaule/turing-screen/internal/domain/service/renderer"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
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

			// Start compositor: collectors + render loop that feeds jobs
			cpuCfg := app.Config.GetCPUSensorConfig()
			gpuCfg := app.Config.GetGPUSensorConfig()
			netCfg := app.Config.GetNetworkConfig()
			diskCfg := app.Config.GetDiskSensorConfig()
			memCfg := app.Config.GetMemoryConfig()
			weatherConfig := app.Config.GetWeatherConfig()

			values := &compositor.SensorValues{}

			var weatherClient *weather.Client
			if weatherConfig.Enabled {
				weatherClient = weather.NewClient()
			}

			compositor.StartCollectors(
				sensorCtx, app.Log, values,
				cpuCfg.TemperatureSensor,
				diskCfg.TemperatureSensor,
				netCfg,
				gpuProvider.NewGPUProvider(gpuCfg.Provider, app.Log),
				weatherClient,
				weatherConfig.City,
				struct {
					CPU, GPU, Memory, Disk, Network time.Duration
					Weather                         time.Duration
				}{
					CPU:     cpuCfg.GetInterval(),
					GPU:     gpuCfg.GetInterval(),
					Memory:  memCfg.GetInterval(),
					Disk:    diskCfg.GetInterval(),
					Network: netCfg.GetInterval(),
					Weather: weatherConfig.GetInterval(),
				},
			)

			comp := compositor.New(
				app.Log, devSerial, builder,
				statsTheme.GetStats(), values,
				cmdUpdate.WithSource("compositor"),
				statsTheme.GetDisplay(),
				200*time.Millisecond,
			)
			comp.SetJobs(jobs)

			sensorWg.Add(1)
			go func() {
				defer sensorWg.Done()
				app.Log.Info("compositor render loop started")
				if err := comp.Run(sensorCtx); err != nil && sensorCtx.Err() == nil {
					app.Log.Errorf("compositor error: %v", err)
				}
			}()

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
			// Flush serial buffer to discard any pending responses
			time.Sleep(100 * time.Millisecond)
			flushBuf := make([]byte, 1024)
			devSerial.Read(flushBuf)

			sensorsRunning = false
			app.Log.Info("all sensors stopped, serial flushed")
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
