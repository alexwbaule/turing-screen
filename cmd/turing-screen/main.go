package main

import (
	"context"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command/brightness"
	"github.com/alexwbaule/turing-screen/internal/domain/command/device"
	"github.com/alexwbaule/turing-screen/internal/domain/command/media"
	"github.com/alexwbaule/turing-screen/internal/domain/command/payload"
	"github.com/alexwbaule/turing-screen/internal/domain/command/update_payload"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	device2 "github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
	"golang.org/x/image/colornames"
	"golang.org/x/sync/errgroup"
	"image/color"
	"math/rand"
	"os"
	"time"
)

func main() {

	app := application.NewApplication()

	jobs := make(chan any)

	app.Run(func(ctx context.Context) error {
		devSerial, err := serial.NewSerial(app.Config.GetDevicePort(), app.Log)
		if err != nil {
			app.Log.Fatal(err.Error())
		}
		worker := sender.NewWorker(ctx, devSerial, app.Log)
		staticImages := app.Theme.GetStaticImages()
		staticTexts := app.Theme.GetStaticTexts()

		cmdDevice := device.NewDevice(app.Log)
		cmdMedia := media.NewMedia(app.Log)
		cmdBright := brightness.NewBrightness(app.Log)
		cmdPayload := payload.NewPayload(app.Log)
		cmdUpdate := update_payload.NewUpdatePayload(app.Log)

		builder := local.NewBuilder(app.Log)

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			_ = worker.Run(0, jobs)
			return devSerial.Close()
		})

		g.Go(func() error {
			<-ctx.Done()
			app.Log.Info(cmdUpdate.GetFPS())
			return nil
		})

		app.Theme.GetCPUStats()

		bg := builder.BuildBackgroundImage(staticImages)
		fbg := builder.BuildBackgroundTexts(bg, staticTexts)
		background := device2.NewImageProcess(fbg)

		jobs <- cmdDevice.Hello()
		jobs <- cmdMedia.StopVideo()
		jobs <- cmdMedia.StopMedia()
		jobs <- cmdBright.SetBrightness(app.Config.GetDeviceDisplay().Brightness)
		jobs <- cmdPayload.SendPayload(background)
		jobs <- cmdMedia.QueryStatus()

		os.Exit(0)
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					app.Log.Errorf("Stopping For fake...")
					close(jobs)
					return context.Canceled
				default:

					V := entity.StatText{
						Font:            utils.DefaultFontFace(),
						Align:           entity.CENTER,
						FontColor:       color.White,
						BackgroundColor: color.Transparent,
						Padding:         4,
						X:               40,
						Y:               66,
					}
					img := builder.DrawText(fbg, fmt.Sprintf("%d%%", rand.Intn(9)), V)
					imgUpdt := device2.NewImageProcess(img)
					jobs <- cmdUpdate.SendPayload(imgUpdt, V.X, V.Y)
					jobs <- cmdMedia.QueryStatus()

					V2 := entity.StatText{
						Font:            utils.DefaultFontFace(),
						Align:           entity.RIGHT,
						FontColor:       colornames.Hotpink,
						BackgroundColor: color.Transparent,
						Padding:         4,
						X:               510,
						Y:               66,
					}
					img2 := builder.DrawText(fbg, fmt.Sprintf("%d%%", rand.Intn(9)), V2)
					imgUpdt2 := device2.NewImageProcess(img2)
					jobs <- cmdUpdate.SendPayload(imgUpdt2, V2.X, V2.Y)
					jobs <- cmdMedia.QueryStatus()

					V3 := entity.StatText{
						Font:            utils.DefaultFontFace(),
						Align:           entity.LEFT,
						FontColor:       colornames.Yellow,
						BackgroundColor: color.Transparent,
						Padding:         4,
						X:               200,
						Y:               361,
					}
					img3 := builder.DrawText(fbg, fmt.Sprintf("%d%%", rand.Intn(9)), V3)
					imgUpdt3 := device2.NewImageProcess(img3)
					jobs <- cmdUpdate.SendPayload(imgUpdt3, V3.X, V3.Y)
					jobs <- cmdMedia.QueryStatus()

				}
				time.Sleep(1000 * time.Millisecond)
			}
		})
		return g.Wait()
	})
}
