package main

import (
	"context"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application"
	"github.com/alexwbaule/turing-screen/internal/application/theme"
	"github.com/alexwbaule/turing-screen/internal/domain/command"

	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	device2 "github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"time"
)

func main() {

	app := application.NewApplication()

	jobs := make(chan any)

	app.Run(func(ctx context.Context) error {

		app.Log.Infof("Device: %+v", app.Config.GetDeviceDisplay())

		devSerial, err := serial.NewSerial(app.Config.GetDevicePort(), app.Log)
		if err != nil {
			app.Log.Fatal(err.Error())
		}

		statsTheme, err := theme.NewTheme(app.Config.GetThemeName(), app.Log)
		if err != nil {
			return err
		}

		worker := sender.NewWorker(ctx, devSerial, app.Log)

		cmdDevice := command.NewDevice(app.Log)
		cmdMedia := command.NewMedia(app.Log)
		cmdBright := command.NewBrightness(app.Log)
		cmdPayload := command.NewPayload(app.Log)
		cmdUpdate := command.NewUpdatePayload(app.Log)

		builder := local.NewBuilder(app.Log)

		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			_ = worker.Run(jobs)
			return devSerial.Close()
		})

		g.Go(func() error {
			<-ctx.Done()
			app.Log.Info(cmdUpdate.GetFPS())
			return nil
		})

		bg := builder.BuildBackgroundImage(statsTheme.GetStaticImages())
		fbg := builder.BuildBackgroundTexts(bg, statsTheme.GetStaticTexts())
		background := device2.NewImageProcess(fbg)

		jobs <- cmdDevice.Hello()
		jobs <- cmdMedia.StopVideo()
		jobs <- cmdMedia.StopMedia()
		jobs <- cmdBright.SetBrightness(app.Config.GetDeviceDisplay().Brightness)
		jobs <- cmdPayload.SendPayload(background)
		jobs <- cmdMedia.QueryStatus()

		stats := statsTheme.GetStats()

		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					app.Log.Errorf("Stopping For fake...")
					close(jobs)
					return context.Canceled
				default:

					if stats.CPU.Temperature.Text != nil {
						V := stats.CPU.Temperature.Text
						img := builder.DrawText(fbg, fmt.Sprintf("%3d°C", rand.Intn(200)), *V)
						imgUpdt := device2.NewImageProcess(img)
						jobs <- cmdUpdate.SendPayload(imgUpdt, V.X, V.Y)
						jobs <- cmdMedia.QueryStatus()
					}

					if stats.CPU.Percentage.Graph != nil {
						V := stats.CPU.Percentage.Graph
						img := builder.DrawProgressBar(fbg, rand.Intn(100), *V)
						imgUpdt := device2.NewImageProcess(img)
						jobs <- cmdUpdate.SendPayload(imgUpdt, V.X, V.Y)
						jobs <- cmdMedia.QueryStatus()
					}

					if stats.GPU.Temperature.Text != nil {
						V2 := stats.GPU.Temperature.Text
						img2 := builder.DrawText(fbg, fmt.Sprintf("%3d%%", rand.Intn(150)), *V2)
						imgUpdt2 := device2.NewImageProcess(img2)
						jobs <- cmdUpdate.SendPayload(imgUpdt2, V2.X, V2.Y)
						jobs <- cmdMedia.QueryStatus()
					}

					if stats.GPU.Percentage.Graph != nil {
						V3 := stats.GPU.Percentage.Graph
						img3 := builder.DrawProgressBar(fbg, rand.Intn(100), *V3)
						imgUpdt3 := device2.NewImageProcess(img3)
						jobs <- cmdUpdate.SendPayload(imgUpdt3, V3.X, V3.Y)
						jobs <- cmdMedia.QueryStatus()
					}

					if stats.Memory.Virtual.PercentText != nil {
						V3 := stats.Memory.Virtual.PercentText
						img3 := builder.DrawText(fbg, fmt.Sprintf("%dMB", rand.Intn(9)), *V3)
						imgUpdt3 := device2.NewImageProcess(img3)
						jobs <- cmdUpdate.SendPayload(imgUpdt3, V3.X, V3.Y)
						jobs <- cmdMedia.QueryStatus()
					}

					if stats.Memory.Virtual.Graph != nil {
						V3 := stats.Memory.Virtual.Graph
						img3 := builder.DrawProgressBar(fbg, rand.Intn(100), *V3)
						imgUpdt3 := device2.NewImageProcess(img3)
						jobs <- cmdUpdate.SendPayload(imgUpdt3, V3.X, V3.Y)
						jobs <- cmdMedia.QueryStatus()
					}

				}
				time.Sleep(500 * time.Millisecond)
			}
		})
		return g.Wait()
	})
}
