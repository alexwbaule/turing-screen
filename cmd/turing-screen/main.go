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
	"golang.org/x/sync/errgroup"
	"image/color"
	"math/rand"
	"os"
)

func main() {

	app := application.NewApplication()

	jobs := make(chan any)

	app.Run(func(ctx context.Context) error {
		devSerial, err := serial.NewSerial(app.Config.GetDevicePort(), app.Log)
		if err != nil {
			app.Log.Error(err.Error())
			os.Exit(-1)
		}
		worker := sender.NewWorker(ctx, devSerial, app.Log)

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			return worker.Run(0, jobs)
		})

		g.Go(func() error {
			<-ctx.Done()
			return devSerial.Close()
		})

		staticImages := app.Theme.GetStaticImages()
		staticTexts := app.Theme.GetStaticTexts()

		cmdDevice := device.NewDevice(app.Log)
		cmdMedia := media.NewMedia(app.Log)
		cmdBright := brightness.NewBrightness(app.Log)
		cmdPayload := payload.NewPayload(app.Log)
		cmdUpdate := update_payload.NewUpdatePayload(app.Log)

		builder := local.NewBuilder(app.Log)

		bg := builder.BuildBackgroundImage(staticImages)
		fbg := builder.BuildBackgroundTexts(bg, staticTexts)
		background := device2.NewImageProcess(fbg)

		jobs <- cmdDevice.Hello()
		jobs <- cmdMedia.StopVideo()
		jobs <- cmdMedia.StopMedia()
		jobs <- cmdBright.SetBrightness(10)
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
					V := entity.StaticText{
						Text:            fmt.Sprintf("%d%%", rand.Intn(100)),
						Font:            utils.DefaultFontFace(),
						FontColor:       color.White,
						BackgroundColor: color.Transparent,
						X:               54,
						Y:               70,
					}

					img := builder.DrawText(fbg, V)

					imgUpdt := device2.NewImageProcess(img)

					_, err = devSerial.Write(cmdUpdate.SendPayload(imgUpdt, V.X, V.Y))
					if err != nil {
						//devSerial.ResetDevice()
						app.Log.Error(err.Error())
						close(jobs)
						//break
						return fmt.Errorf("stop")
					}

					_, err = devSerial.Write(cmdMedia.QueryStatus())
					if err != nil {
						//devSerial.ResetDevice()
						app.Log.Error(err.Error())
						close(jobs)
						//break
						return fmt.Errorf("stop")
					}
				}
				//time.Sleep(100 * time.Millisecond)
			}
		})
		return g.Wait()
	})
}
