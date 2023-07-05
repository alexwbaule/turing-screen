package main

import (
	"context"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application"
	"golang.org/x/sync/errgroup"
)

func main() {

	app := application.NewApplication()

	app.Run(func(ctx context.Context) error {
		fmt.Printf("OK\n")
		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			fmt.Printf("OK\n")
			fmt.Printf("[%#v]\n", app.Theme.GetGPUStats())

			return nil
		})
		return g.Wait()

		/*
			portName := cfg.GetString("device.port")
			themeName := cfg.GetString("device.theme")

			//g, ctx := errgroup.WithContext(ctx)

			staticImages := themeConf.GetStaticImages()
			staticTexts := themeConf.GetStaticTexts()
			log.Infof("Orientation: %s", themeConf.GetOrientation())

			themeConf.GetCPUStats()
			themeConf.GetGPUStats()
			themeConf.GetDiskStats()
			themeConf.GetMemoryStats()
			themeConf.GetNetworkStats()

			devSerial, err := serial.NewSerial(portName, log)
			if err != nil {
				log.Error(err.Error())
				os.Exit(-1)
			}

			cmdDevice := cmddevice.NewDevice(log)
			cmdMedia := media.NewMedia(log)
			cmdBright := brightness.NewBrightness(log)
			cmdPayload := payload.NewPayload(log)
			cmdUpdate := update_payload.NewUpdatePayload(log)

			bg := local.BuildBackgroundImage(staticImages)

			fbg := local.BuildBackgroundTexts(bg, staticTexts)
			background := device.NewImageProcess(fbg)

			//os.Exit(0)

			_, err = devSerial.Write(cmdDevice.Hello())
			if err != nil {
				devSerial.ResetDevice()
				log.Error(err.Error())
			}
			_, err = devSerial.Write(cmdMedia.StopVideo())
			if err != nil {
				devSerial.ResetDevice()
				log.Error(err.Error())
			}
			_, err = devSerial.Write(cmdMedia.StopMedia())
			if err != nil {
				devSerial.ResetDevice()
				log.Error(err.Error())
			}

			_, err = devSerial.Write(cmdBright.SetBrightness(10))
			if err != nil {
				devSerial.ResetDevice()
				log.Error(err.Error())
			}

			_, err = devSerial.Write(cmdPayload.SendPayload(background))
			if err != nil {
				devSerial.ResetDevice()
				log.Error(err.Error())
			}

			_, err = devSerial.Write(cmdMedia.QueryStatus())
			if err != nil {
				devSerial.ResetDevice()
				log.Error(err.Error())
			}

			imgId := 0

			for {
				V := entity.StaticText{
					Text:            fmt.Sprintf("%d%%", rand.Intn(100)),
					Font:            utils.DefaultFontFace(),
					FontColor:       color.White,
					BackgroundColor: color.Transparent,
					X:               54,
					Y:               70,
				}

				img := local.DrawText(fbg, V)

				imgUpdt := device.NewImageProcess(img)

				_, err = devSerial.Write(cmdUpdate.SendPayload(imgUpdt, V.X, V.Y))
				if err != nil {
					//devSerial.ResetDevice()
					log.Error(err.Error())
					//break
				}

				_, err = devSerial.Write(cmdMedia.QueryStatus())
				if err != nil {
					//devSerial.ResetDevice()
					log.Error(err.Error())
					break
				}
				if imgId == 100 {
					imgId = 0
					continue
				}
				imgId++
				//time.Sleep(100 * time.Millisecond)
			}
			/*
				_, err = devSerial.Write(cmdDevice.TurnOff())
				if err != nil {
					devSerial.ResetDevice()
					log.Error(err.Error())
				}
		*/

	})
}
