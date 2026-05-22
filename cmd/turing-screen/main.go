package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application"
	"github.com/alexwbaule/turing-screen/internal/application/hwinfo"
	apptheme "github.com/alexwbaule/turing-screen/internal/application/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sensors"
	gpuprovider "github.com/alexwbaule/turing-screen/internal/resource/gpu"
	device2 "github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
	"golang.org/x/sync/errgroup"
)

// sleepCommand is a no-op command that sleeps for a duration when processed.
type sleepCommand struct {
	duration time.Duration
}

func (s *sleepCommand) GetBytes() [][]byte {
	time.Sleep(s.duration)
	return nil
}
func (s *sleepCommand) GetName() string                        { return "SLEEP" }
func (s *sleepCommand) ValidateWrite() command.WriteValidation { return command.WriteValidation{} }
func (s *sleepCommand) ValidateCommand([]byte, int) error      { return nil }
func (s *sleepCommand) SetCount(int64)                         {}

func main() {
	app := application.NewApplication()

	app.Run(func(ctx context.Context) error {
		app.Log.Infof("device display: %#v", app.Config.GetDeviceDisplay())

		devSerial, err := serial.NewSerial(app.Config.GetDevicePort(), app.Log)
		if err != nil {
			app.Log.Fatal(err.Error())
		}

		statsTheme, err := apptheme.NewTheme(app.Config, app.Log)
		if err != nil {
			return err
		}

		// Detect hardware info and replace template placeholders
		hw := hwinfo.Detect(app.Log, app.Config.GetGPUSensorConfig().Provider)
		staticTexts := statsTheme.GetStaticTexts()
		for key, st := range staticTexts {
			st.Text = hw.ReplaceText(st.Text)
			staticTexts[key] = st
		}

		builder := local.NewBuilder(app.Log, app.Config.GetDeviceDisplay(), statsTheme.GetDisplay())
		bg := builder.BuildBackgroundImage(statsTheme.GetStaticImages())
		fbg := builder.BuildBackgroundTexts(bg, staticTexts)
		background := device2.NewImageProcess(fbg)

		cmdDevice := command.NewDevice(app.Log)
		cmdMedia := command.NewMedia(app.Log)
		cmdBright := command.NewBrightness(app.Log)
		cmdPayload := command.NewPayload(app.Log, statsTheme.GetDisplay().Orientation)
		cmdUpdate := command.NewUpdatePayload(app.Log, statsTheme.GetDisplay().Orientation, app.Config.GetDeviceDisplay())
		cmdPreUpdate := command.NewPreUpdateBitmap(app.Log)
		cmdHealthCheck := command.NewHealthCheck(app.Log)
		cmdStorage := command.NewStorage(app.Log)
		encoding := command.EncodingBGR
		queue := sender.NewRegionQueue(128)
		worker := sender.NewWorker(ctx, devSerial, background, cmdDevice, cmdMedia, cmdPayload, cmdPreUpdate, cmdHealthCheck, app.Log, queue)

		videoEntries := statsTheme.GetVideoPlay()
		stats := statsTheme.GetStats()

		app.Log.Info("starting app")

		if len(videoEntries) > 0 {
			// ===== FLUXO 2: VÍDEO =====
			// The overlay uses BLACK (0,0,0) as transparent — the device shows
			// the video through black pixels. Sensor data renders on black background.
			// Create a black background for the overlay.
			blackBg := builder.BuildTransparentBackground() // RGBA all zeros = black+transparent
			blackFbg := builder.BuildBackgroundTexts(blackBg, staticTexts)
			videoBackground := device2.NewImageProcess(blackFbg)

			// For video mode, sensor updates use BGR (same as static mode)
			// The device handles compositing internally

			// Clear all BackgroundImage references in stats so sensors render
			// on transparent/black background (video shows through)
			clearBackgroundImages(stats)
			// Pré-fluxo roda SÍNCRONO (direto no serial, antes do worker)

			// Determinar paths
			var videoLocalPath, videoDevicePath string
			for _, vid := range videoEntries {
				videoLocalPath = vid.Path
				videoDevicePath = "/root/video/" + filepath.Base(vid.Path)
				break
			}
			app.Log.Infof("video theme: local=%s device=%s", videoLocalPath, videoDevicePath)

			// --- PRÉ-FLUXO SÍNCRONO ---
			if err := videoPreFlow(devSerial, cmdDevice, cmdMedia, cmdBright, cmdStorage,
				app.Config.GetDeviceDisplay().Brightness, videoLocalPath, videoDevicePath, app.Log); err != nil {
				return fmt.Errorf("video pre-flow failed: %w", err)
			}

			// --- FLUXO PRINCIPAL (via queue) ---
			queue.Enqueue(cmdStorage.RestartDevice())
			queue.Enqueue(&sleepCommand{duration: 2 * time.Second})
			queue.Enqueue(cmdDevice.Hello())
			queue.Enqueue(cmdStorage.GetFileInfo(videoDevicePath))
			queue.Enqueue(cmdStorage.PlayVideo(videoDevicePath, true))
			queue.Enqueue(cmdPreUpdate)
			queue.Enqueue(cmdBright.SetBrightness(app.Config.GetDeviceDisplay().Brightness))
			queue.Enqueue(cmdPayload.SendOverlay(videoBackground))

		} else {
			// ===== FLUXO 1: IMAGEM ESTÁTICA =====
			queue.Enqueue(cmdDevice.Hello())
			queue.Enqueue(cmdMedia.StopVideo())
			queue.Enqueue(cmdMedia.StopMedia())
			queue.Enqueue(cmdBright.SetBrightness(app.Config.GetDeviceDisplay().Brightness))
			queue.Enqueue(cmdPreUpdate)
			queue.Enqueue(cmdPayload.SendPayload(background))
		}

		// ===== START WORKER + SENSORS =====
		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			app.Log.Info("starting reader worker")
			return worker.Run()
		})

		g.Go(func() error {
			<-ctx.Done()
			app.Log.Info("shutdown device")
			_ = worker.OffChannel(cmdDevice.TurnOff())
			app.Log.Infof("cleaning queue with %d entries", queue.Len())
			count := 0
			timeout := time.After(5 * time.Second)
			for {
				select {
				case <-timeout:
					goto done
				default:
					_, ok := queue.Dequeue()
					if !ok {
						goto done
					}
					count++
				}
			}
		done:
			app.Log.Infof("drained %d messages from queue", count)
			_ = devSerial.Close()
			return ctx.Err()
		})

		cpu := sensors.NewCpuStat(app.Log, queue, builder, cmdUpdate, encoding, app.Config.GetCPUSensorConfig().TemperatureSensor)
		mem := sensors.NewMemStat(app.Log, queue, builder, cmdUpdate, encoding)
		dt := sensors.NewDateTimeStat(app.Log, queue, builder, cmdUpdate, encoding)
		net := sensors.NewDNetStat(app.Log, queue, builder, cmdUpdate, app.Config.GetNetworkConfig(), encoding)
		dsk := sensors.NewDiskStat(app.Log, queue, builder, cmdUpdate, encoding, app.Config.GetDiskSensorConfig().TemperatureSensor)
		gpu := sensors.NewGpuStat(app.Log, queue, builder, cmdUpdate, encoding, gpuprovider.NewGPUProvider(app.Config.GetGPUSensorConfig().Provider, app.Log))

		if stats.CPU.Percentage != nil {
			g.Go(func() error { return cpu.RunPercentage(ctx, stats.CPU.Percentage) })
		}
		if stats.CPU.Frequency != nil {
			g.Go(func() error { return cpu.RunFrequency(ctx, stats.CPU.Frequency) })
		}
		if stats.CPU.Temperature != nil {
			g.Go(func() error { return cpu.RunTemperature(ctx, stats.CPU.Temperature) })
		}
		if stats.Memory != nil {
			g.Go(func() error { return mem.RunMemStat(ctx, stats.Memory) })
		}
		if stats.Date != nil {
			g.Go(func() error { return dt.RunDateTime(ctx, stats.Date) })
		}
		if stats.Net != nil {
			g.Go(func() error { return net.RunNetStat(ctx, stats.Net) })
		}
		if stats.Disk != nil {
			g.Go(func() error { return dsk.RunDiskStat(ctx, stats.Disk) })
		}
		if stats.GPU != nil {
			g.Go(func() error { return gpu.RunGpuStat(ctx, stats.GPU) })
		}

		return g.Wait()
	})
}

// videoPreFlow runs the synchronous pre-flow for video themes.
// This runs BEFORE the worker starts, using the serial port directly.
// It handles: init, check if video exists, upload if needed.
func videoPreFlow(
	ser *serial.Serial,
	cmdDevice *command.Device,
	cmdMedia *command.Media,
	cmdBright *command.Brightness,
	cmdStorage *command.Storage,
	brightness int,
	localPath, devicePath string,
	log interface {
		Infof(string, ...interface{})
		Errorf(string, ...interface{})
	},
) error {
	log.Infof("video pre-flow: init + check/upload")

	// HELLO
	log.Infof(">>> HELLO")
	if _, err := ser.Write(cmdDevice.Hello()); err != nil {
		return fmt.Errorf("HELLO failed: %w", err)
	}
	log.Infof("<<< HELLO ok")

	// STOP_VIDEO
	log.Infof(">>> STOP_VIDEO")
	if _, err := ser.Write(cmdMedia.StopVideo()); err != nil {
		return fmt.Errorf("STOP_VIDEO failed: %w", err)
	}
	log.Infof("<<< STOP_VIDEO ok")

	// STOP_MEDIA
	log.Infof(">>> STOP_MEDIA")
	if _, err := ser.Write(cmdMedia.StopMedia()); err != nil {
		return fmt.Errorf("STOP_MEDIA failed: %w", err)
	}
	log.Infof("<<< STOP_MEDIA ok")

	// SET_BRIGHTNESS
	log.Infof(">>> SET_BRIGHTNESS (%d)", brightness)
	if _, err := ser.Write(cmdBright.SetBrightness(brightness)); err != nil {
		return fmt.Errorf("SET_BRIGHTNESS failed: %w", err)
	}
	log.Infof("<<< SET_BRIGHTNESS ok")

	// CMD_0x7D (pre-upload setup) — only needed if upload will happen
	// Skipping for now since we check file existence first
	// log.Infof(">>> CMD_0x7D (pre-upload)")
	// if _, err := ser.Write(cmdStorage.SetPreUpload(byte(brightness))); err != nil {
	// 	return fmt.Errorf("SET_PRE_UPLOAD failed: %w", err)
	// }
	// log.Infof("<<< CMD_0x7D ok")

	// GET_STORAGE_STATUS
	log.Infof(">>> GET_STORAGE_STATUS")
	if _, err := ser.Write(cmdStorage.GetStorageStatus()); err != nil {
		return fmt.Errorf("GET_STORAGE_STATUS failed: %w", err)
	}
	log.Infof("<<< GET_STORAGE_STATUS ok")

	// GET_FILE_INFO — check if video exists on device
	log.Infof(">>> GET_FILE_INFO %s", devicePath)
	if _, err := ser.Write(cmdStorage.GetFileInfo(devicePath)); err != nil {
		return fmt.Errorf("GET_FILE_INFO failed: %w", err)
	}
	log.Infof("<<< GET_FILE_INFO ok")

	// TODO: Parse the GET_FILE_INFO response to check if file exists.
	// For now, assume the video is already on the device.
	// If file doesn't exist, we need to upload it:
	//
	// Upload sequence:
	//   ser.Write(cmdMedia.StopVideo())
	//   ser.Write(cmdMedia.StopMedia())
	//   ser.Write(cmdStorage.CreateFile(devicePath, fileSize))
	//   ser.WriteRaw(fileData)
	//   ser.ReadPoll(60 * time.Second) → "file_rev_done"
	//
	// The upload works in Python (scripts/upload_video.py).
	// To implement in Go, need to:
	// 1. Read the response from GET_FILE_INFO (parse "0" vs size)
	// 2. If "0": read local file, send CREATE_FILE, WriteRaw, ReadPoll
	// 3. The file must be in device format (H.264 800x480 24fps MP4)

	_ = localPath
	_ = os.ReadFile // will be used for upload

	log.Infof("video pre-flow: complete (assuming video on device)")
	return nil
}

// clearBackgroundImages removes all BackgroundImage references from stats
// so that sensor updates render on transparent/black background.
// This is required for video mode where black = transparent (video shows through).
func clearBackgroundImages(stats *theme.Stats) {
	// Use a simple approach: nil out BackgroundImage in all known text/graph fields
	// This makes sensors render on black background (transparent for video overlay)
	if stats.CPU != nil {
		if stats.CPU.Percentage != nil {
			if stats.CPU.Percentage.Text != nil {
				stats.CPU.Percentage.Text.BackgroundImage = nil
			}
			if stats.CPU.Percentage.Graph != nil {
				stats.CPU.Percentage.Graph.BackgroundImage = nil
			}
		}
		if stats.CPU.Temperature != nil {
			if stats.CPU.Temperature.Text != nil {
				stats.CPU.Temperature.Text.BackgroundImage = nil
			}
		}
		if stats.CPU.Frequency != nil {
			if stats.CPU.Frequency.Text != nil {
				stats.CPU.Frequency.Text.BackgroundImage = nil
			}
		}
	}
	if stats.GPU != nil {
		if stats.GPU.Percentage != nil {
			if stats.GPU.Percentage.Text != nil {
				stats.GPU.Percentage.Text.BackgroundImage = nil
			}
			if stats.GPU.Percentage.Graph != nil {
				stats.GPU.Percentage.Graph.BackgroundImage = nil
			}
		}
		if stats.GPU.Temperature != nil {
			if stats.GPU.Temperature.Text != nil {
				stats.GPU.Temperature.Text.BackgroundImage = nil
			}
		}
	}
	if stats.Memory != nil {
		if stats.Memory.Virtual != nil && stats.Memory.Virtual.PercentText != nil {
			stats.Memory.Virtual.PercentText.BackgroundImage = nil
		}
	}
	if stats.Disk != nil {
		if stats.Disk.Used != nil && stats.Disk.Used.Percent != nil {
			stats.Disk.Used.Percent.BackgroundImage = nil
		}
	}
}
