package main

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/config"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/theme"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/command/brightness"
	cmddevice "github.com/alexwbaule/turing-screen/internal/domain/command/device"
	"github.com/alexwbaule/turing-screen/internal/domain/command/media"
	"github.com/alexwbaule/turing-screen/internal/domain/command/payload"
	"github.com/alexwbaule/turing-screen/internal/domain/command/update_payload"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/process/local"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
	"image/color"
	"math/rand"
	"os"
	"time"
)

func main() {
	log := logger.NewLogger()

	log.Info("Starting application")

	cfg, err := config.NewDefaultConfig()
	if err != nil {
		log.Errorf("error opening config (%s): %s", err)
		return
	}
	portName := cfg.GetString("device.port")
	themeName := cfg.GetString("device.theme")

	log.Infof("Opening Theme: %s", themeName)
	themeConf, err := theme.LoadTheme(themeName)
	if err != nil {
		log.Errorf("error opening theme (%s): %s", themeName, err)
		return
	}
	staticImages := themeConf.GetStaticImages()

	for s, images := range staticImages {
		fmt.Printf("Background:[%s] [%+v]\n", s, images)
	}

	staticTexts := themeConf.GetStaticTexts()
	for s, texts := range staticTexts {
		fmt.Printf("Texts:[%s] [%+v]\n", s, texts)
	}

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
		V := entity.StaticTexts{
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
		time.Sleep(100 * time.Millisecond)
	}
	/*
		_, err = devSerial.Write(cmdDevice.TurnOff())
		if err != nil {
			devSerial.ResetDevice()
			log.Error(err.Error())
		}
	*/
}
