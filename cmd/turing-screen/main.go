package main

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/command/brightness"
	cmddevice "github.com/alexwbaule/turing-screen/command/device"
	"github.com/alexwbaule/turing-screen/command/media"
	"github.com/alexwbaule/turing-screen/command/payload"
	"github.com/alexwbaule/turing-screen/command/update_payload"
	"github.com/alexwbaule/turing-screen/config"
	"github.com/alexwbaule/turing-screen/device_serial"
	"github.com/alexwbaule/turing-screen/image_process"
	"github.com/alexwbaule/turing-screen/logger"
	"github.com/alexwbaule/turing-screen/utils"
	"github.com/disintegration/gift"
	"github.com/fogleman/gg"
	"image"
	"os"
	"time"
)

func main() {
	log := logger.NewLogger()

	log.Info("Starting application")

	cfg, err := config.NewConfig(nil)
	if err != nil {
		return
	}
	portName := cfg.GetString("device.port")

	for {
		devSerial, err := device_serial.NewSerial(portName, log)
		if err != nil {
			log.Error(err.Error())
			os.Exit(-1)
		}

		cmdDevice := cmddevice.NewDevice()
		cmdMedia := media.NewMedia()
		cmdBright := brightness.NewBrightness()
		cmdPayload := payload.NewPayload()
		cmdUpdate := update_payload.NewUpdatePayload()

		bg := utils.LoadImage("res/backgrounds/example5inch_landscape.png")

		background := image_process.NewImageProcess(bg)

		_, err = devSerial.Write(cmdDevice.Hello())
		if err != nil {
			devSerial.ResetDevice()
			log.Error(err.Error())
			continue
		}
		_, err = devSerial.Write(cmdMedia.StopVideo())
		if err != nil {
			devSerial.ResetDevice()
			log.Error(err.Error())
			continue
		}
		_, err = devSerial.Write(cmdMedia.StopMedia())
		if err != nil {
			devSerial.ResetDevice()
			log.Error(err.Error())
			continue
		}

		_, err = devSerial.Write(cmdBright.SetBrightness(10))
		if err != nil {
			devSerial.ResetDevice()
			log.Error(err.Error())
			continue
		}

		_, err = devSerial.Write(cmdPayload.SendPayload(background))
		if err != nil {
			devSerial.ResetDevice()
			log.Error(err.Error())
			continue
		}

		_, err = devSerial.Write(cmdMedia.QueryStatus())
		if err != nil {
			devSerial.ResetDevice()
			log.Error(err.Error())
			continue
		}

		imgId := 1
		times := 0
		for {
			ctx := gg.NewContextForImage(bg)
			numb := utils.LoadImage(fmt.Sprintf("res/test/n%d.png", imgId))

			x := 660
			y := 340

			ctx.DrawImage(numb, x, y)
			ii := ctx.Image()
			crp := image.Rect(x, y, 140+x, 140+y)

			g := gift.New(
				gift.Crop(crp),
			)
			dst := image.NewRGBA(image.Rect(0, 0, 140, 140))

			g.Draw(dst, ii)

			imgUpdt := image_process.NewImageProcess(dst)

			_, err = devSerial.Write(cmdUpdate.SendPayload(imgUpdt, x, y))
			if err != nil {
				//devSerial.ResetDevice()
				log.Error(err.Error())
				//break
			}

			_, err = devSerial.Write(cmdMedia.QueryStatus())
			if err != nil {
				//devSerial.ResetDevice()
				log.Error(err.Error())
				//break
			}
			time.Sleep(1 * time.Second)

			if times == 100000 {
				break
			}
			if imgId == 3 {
				imgId = 1
				continue
			}
			imgId++
			times++
		}
		//_, err = devSerial.Write(cmdDevice.TurnOff())
		//if err != nil {
		//devSerial.ResetDevice()
		//log.Error(err.Error())
		//time.Sleep(5 * time.Second)
		//continue
		//}
		time.Sleep(5 * time.Second)
	}
}
