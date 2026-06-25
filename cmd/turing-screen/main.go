package main

import (
	"context"
	"image"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"sync/atomic"
	"time"

	pyroscope "github.com/grafana/pyroscope-go"

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
	"github.com/alexwbaule/turing-screen/internal/resource/turzx"
	"github.com/alexwbaule/turing-screen/internal/resource/volume"
	"github.com/alexwbaule/turing-screen/internal/resource/weather"
	"golang.org/x/sync/errgroup"
)

func startProfiling(appName, pprofAddr string) {
	// pprof — always available, zero external deps
	go func() {
		log.Printf("[profiling] pprof listening on http://%s/debug/pprof/", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Printf("[profiling] pprof error: %v", err)
		}
	}()

	// Pyroscope continuous profiling — only when PYROSCOPE_ADDR is set
	if addr := os.Getenv("PYROSCOPE_ADDR"); addr != "" {
		_, err := pyroscope.Start(pyroscope.Config{
			ApplicationName: appName,
			ServerAddress:   addr,
			Logger:          pyroscope.StandardLogger,
			ProfileTypes: []pyroscope.ProfileType{
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,
				pyroscope.ProfileGoroutines,
			},
		})
		if err != nil {
			log.Printf("[profiling] pyroscope start error: %v", err)
		} else {
			log.Printf("[profiling] pyroscope pushing to %s as '%s'", addr, appName)
		}
	}
}

func main() {
	startProfiling("turing-screen", "localhost:6060")
	app := application.NewApplication()

	app.Run(func(ctx context.Context) error {
		app.Log.Infof("device display: %#v", app.Config.GetDeviceDisplay())

		var currentValues atomic.Pointer[compositor.SensorValues]

		// --- Device detection: TURZX (USB bulk) takes priority over Rev C (serial) ---
		turzxDrv, turzxErr := turzx.NewDriver(app.Log)
		isTURZX := turzxErr == nil && turzxDrv != nil
		if isTURZX {
			app.Log.Infof("TURZX device detected: PID=0x%04x, native %dx%d", uint16(turzxDrv.PID), turzxDrv.Width, turzxDrv.Height)
			if info, err := turzxDrv.GetStorageInfo(); err != nil {
				app.Log.Warnf("TURZX: storage probe failed: %v", err)
			} else {
				app.Log.Infof("TURZX: storage: %s", info)
			}
		} else {
			app.Log.Infof("TURZX not found (%v), using Rev C serial", turzxErr)
		}

		var devSerial serial.SerialSender
		if !isTURZX {
			var err error
			devSerial, err = serial.NewSerial(app.Config.GetDevicePort(), app.Log)
			if err != nil {
				log.Fatalf("failed to create serial sender: %v", err)
			}
		}

		cmdDevice := command.NewDevice(app.Log)
		cmdBright := command.NewBrightness(app.Log)
		cmdPayload := command.NewPayload(app.Log, 2) // overridden per theme

		// Shared jobs channel (Rev C only; TURZX sends frames directly)
		jobs := make(chan command.Command, 2000)

		var sensorCancel context.CancelFunc
		var sensorWg sync.WaitGroup
		var sensorMu sync.Mutex
		sensorsRunning := false

		var startSensors func()
		var stopSensors func()

		apiController := api.NewDaemonController(
			app.Log, devSerial, cmdDevice, cmdBright, cmdPayload,
			app.Config.GetThemeName(), "chs_5inch",
		)
		if isTURZX {
			apiController.SetTURZXDriver(turzxDrv)
		}
		apiController.SetSensorControl(func() { stopSensors() }, func() { startSensors() })
		apiController.SetSensorFunc(func() map[string]interface{} {
			v := currentValues.Load()
			if v == nil {
				return map[string]interface{}{}
			}
			return v.Snapshot()
		})

		// startSensors initializes theme + sensors and runs them in background goroutines.
		startSensors = func() {
			sensorMu.Lock()
			defer sensorMu.Unlock()
			if sensorsRunning {
				return
			}

			if err := app.Config.Reload(); err != nil {
				app.Log.Warnf("failed to reload config: %v", err)
			}

			statsTheme, err := appTheme.NewTheme(app.Config, app.Log)
			if err != nil {
				app.Log.Errorf("failed to load theme: %v", err)
				apiController.NotifySensorsFailed()
				return
			}

			hw := hwinfo.Detect(app.Log, app.Config.GetGPUSensorConfig().Provider)
			staticTexts := statsTheme.GetStaticTexts()
			for key, st := range staticTexts {
				st.Text = hw.ReplaceText(st.Text)
				staticTexts[key] = st
			}

			orientation := statsTheme.GetDisplay().Orientation
			apiController.SetOrientation(orientation)
			builder := renderer.NewBuilder(app.Log, app.Config.GetDeviceDisplay(), statsTheme.GetDisplay())
			builder.BuildBackgroundImage(statsTheme.GetStaticImages())
			builder.BuildBackgroundTexts(statsTheme.GetStaticTexts())

			values := &compositor.SensorValues{}
			currentValues.Store(values)

			var sensorCtx context.Context
			sensorCtx, sensorCancel = context.WithCancel(ctx)

			var comp *compositor.Compositor

			if isTURZX {
				// TURZX pipeline: initialize device, then full-frame callback.
				brightness := app.Config.GetDeviceDisplay().Brightness
				if err := turzxDrv.Initialize(brightness); err != nil {
					app.Log.Errorf("TURZX init failed: %v", err)
					apiController.NotifySensorsFailed()
					return
				}

				// Start H.264 background video if the theme defines one.
				vp := statsTheme.GetVideoPlay()
				app.Log.Infof("TURZX: theme video section: present=%v", vp != nil)
				if vp != nil {
					app.Log.Infof("TURZX: video bytes=%d path=%s", len(vp.Video), vp.Path)
				}
				if vp != nil && len(vp.Video) > 0 {
					app.Log.Infof("TURZX: starting H264 video stream (%d B)", len(vp.Video))
					turzxDrv.StartVideoStream(sensorCtx, vp.Video)
				}

				comp = compositor.New(
					app.Log, nil, builder,
					statsTheme.GetStats(), values,
					nil,
					statsTheme.GetDisplay(),
					500*time.Millisecond,
				)
				comp.SetFrameCallback(func(frame *image.NRGBA) {
					if err := turzxDrv.SendFrame(frame, orientation); err != nil {
						app.Log.Warnf("TURZX SendFrame: %v", err)
					}
				})
			} else {
				// Rev C pipeline: serial worker + overlay + dirty-rect diff.
				cmdPayload = command.NewPayload(app.Log, orientation)
				background := device.NewImageProcess(builder.GetBackground())

				cmdMedia := command.NewMedia(app.Log)
				cmdStorage := command.NewStorage(app.Log)
				cmdOption := command.NewOption(app.Log)
				cmdUpdate := command.NewUpdatePayload(app.Log, orientation, app.Config.GetDeviceDisplay())

				initService := initializer.New(devSerial, app.Log, app.Config, statsTheme, cmdDevice, cmdMedia, cmdOption, cmdStorage, cmdBright, cmdPayload)
				if err := initService.Run(background); err != nil {
					app.Log.Warnf("device init failed, attempting reconnect: %v", err)
					if rerr := devSerial.ResetDevice(); rerr != nil {
						app.Log.Errorf("device reconnect failed: %v", rerr)
						apiController.NotifySensorsFailed()
						return
					}
					if err := initService.Run(background); err != nil {
						app.Log.Errorf("device init failed after reconnect: %v", err)
						apiController.NotifySensorsFailed()
						return
					}
				}

				if overlay := initService.Overlay(); overlay != nil {
					cmdUpdate.SetOverlay(overlay)
				}

				worker := sender.NewWorker(devSerial, background, cmdDevice, cmdMedia, cmdPayload, app.Log)
				sensorWg.Add(1)
				go func() {
					defer sensorWg.Done()
					if err := worker.Run(sensorCtx, jobs); err != nil && sensorCtx.Err() == nil {
						app.Log.Errorf("worker error: %v", err)
					}
				}()

				if overlay := initService.Overlay(); overlay != nil {
					sensorWg.Add(1)
					go func() {
						defer sensorWg.Done()
						ticker := time.NewTicker(500 * time.Millisecond)
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

				comp = compositor.New(
					app.Log, devSerial, builder,
					statsTheme.GetStats(), values,
					cmdUpdate.WithSource("compositor"),
					statsTheme.GetDisplay(),
					500*time.Millisecond,
				)
				comp.SetJobs(jobs)
			}

			// Common: start sensor collectors
			cpuCfg := app.Config.GetCPUSensorConfig()
			gpuCfg := app.Config.GetGPUSensorConfig()
			netCfg := app.Config.GetNetworkConfig()
			diskCfg := app.Config.GetDiskSensorConfig()
			memCfg := app.Config.GetMemoryConfig()
			weatherConfig := app.Config.GetWeatherConfig()

			var weatherClient *weather.Client
			if weatherConfig.Enabled {
				weatherClient = weather.NewClient()
			}

			var volClient *volume.Client
			if vc, err := volume.NewClient(); err != nil {
				app.Log.Warnf("volume client unavailable: %v", err)
			} else {
				volClient = vc
			}

			compositor.StartCollectors(
				sensorCtx, app.Log, values,
				cpuCfg.TemperatureSensor,
				diskCfg.TemperatureSensor,
				netCfg,
				gpuProvider.NewGPUProvider(gpuCfg.Provider, app.Log),
				weatherClient,
				weatherConfig.City,
				volClient,
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

			sensorWg.Add(1)
			go func() {
				defer sensorWg.Done()
				app.Log.Info("compositor render loop started")
				if err := comp.Run(sensorCtx); err != nil && sensorCtx.Err() == nil {
					app.Log.Errorf("compositor error: %v", err)
				}
			}()

			sensorsRunning = true
			apiController.NotifySensorsStarted()
			app.Log.Info("all sensors started")
		}

		stopSensors = func() {
			sensorMu.Lock()
			defer sensorMu.Unlock()
			if !sensorsRunning {
				return
			}
			if sensorCancel != nil {
				sensorCancel()
			}
			sensorWg.Wait()

			if !isTURZX {
				// Drain remaining jobs
				for {
					select {
					case <-jobs:
					default:
						goto drained
					}
				}
			drained:
				if err := devSerial.Flush(); err != nil {
					app.Log.Warnf("serial flush after stop: %v", err)
				}
			}

			sensorsRunning = false
			app.Log.Info("all sensors stopped")
		}

		g, ctx := errgroup.WithContext(ctx)

		apiServer := api.NewServer(app.Log, apiController, app.Config.GetAPIPort())
		g.Go(func() error {
			return apiServer.Start(ctx)
		})

		g.Go(func() error {
			<-ctx.Done()
			app.Log.Info("shutdown signal received.")
			stopSensors()
			if isTURZX {
				turzxDrv.Close()
			} else {
				if app.Config.GetTurnOffOnExit() {
					app.Log.Info("turning off device.")
					devSerial.Execute(cmdDevice.TurnOff())
				}
				_ = devSerial.Close()
			}
			return nil
		})

		startSensors()

		return g.Wait()
	})
}
