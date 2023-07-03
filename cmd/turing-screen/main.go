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
	"github.com/alexwbaule/turing-screen/logger"
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

		_, err = devSerial.Write(cmdPayload.SendPayload("res/backgrounds/example5inch_landscape.png"))
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
			_, err = devSerial.Write(cmdUpdate.SendPayload(fmt.Sprintf("res/test/n%d.png", imgId)))
			if err != nil {
				devSerial.ResetDevice()
				log.Error(err.Error())
				break
			}

			_, err = devSerial.Write(cmdMedia.QueryStatus())
			if err != nil {
				devSerial.ResetDevice()
				log.Error(err.Error())
				break
			}
			if imgId == 3 {
				imgId = 1
				continue
			}
			if times == 100000 {
				break
			}
			imgId++
			times++
		}
		_, err = devSerial.Write(cmdDevice.TurnOff())
		if err != nil {
			devSerial.ResetDevice()
			log.Error(err.Error())
			time.Sleep(5 * time.Second)
			continue
		}
		time.Sleep(5 * time.Second)
	}
}
