package main

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/command/option"
	"github.com/alexwbaule/turing-screen/config"
	"github.com/alexwbaule/turing-screen/device_serial"
	"github.com/alexwbaule/turing-screen/logger"
	"github.com/alexwbaule/turing-screen/usb"
	"os"
)

func main() {
	log := logger.NewLogger()

	log.Info("Starting application")

	cfg, err := config.NewConfig(nil)
	if err != nil {
		return
	}
	port := cfg.GetString("device.port")

	v := option.SetOptions(option.Default, option.NoFlip, option.Disabled)

	h := v.GetBytes()
	fmt.Printf("%d - [%+v]\n", len(h), h)

	device, err := usb.NewUsbDevice(port, log)
	if err != nil {
		log.Error(err.Error())
		os.Exit(-1)
	}

	devSerial, err := device_serial.NewSerial(device, log)
	if err != nil {
		log.Error(err.Error())
		os.Exit(-1)
	}

	err = devSerial.SendHello()
	if err != nil {
		log.Error(err.Error())
		device.ResetDevice()
		os.Exit(-1)
	}
	err = devSerial.SendStopMedia()
	if err != nil {
		log.Error(err.Error())
		device.ResetDevice()
		os.Exit(-1)
	}

}
