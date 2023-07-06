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
	"github.com/alexwbaule/turing-screen/internal/domain/service/fake"
	"github.com/alexwbaule/turing-screen/internal/domain/service/sender"
	device2 "github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
	"golang.org/x/sync/errgroup"
	"image/color"
	"math/rand"
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

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			_ = worker.Run(0, jobs)
			return devSerial.Close()
		})

		g.Go(func() error {
			<-ctx.Done()
			return nil
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

		fk := fake.NewFakeStat(ctx, app.Log, jobs)

		fk.Run()
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
					jobs <- cmdUpdate.SendPayload(imgUpdt, V.X, V.Y)
					jobs <- cmdMedia.QueryStatus()
				}
				//time.Sleep(100 * time.Millisecond)
			}
		})
		return g.Wait()
	})
}
