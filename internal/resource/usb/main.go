package usb

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/google/gousb"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
	"strconv"
	"time"
)

type UsbDevice struct {
	ProducId     uint16
	VendorId     uint16
	SerialNumber string
	Name         string
	log          *logger.Logger
}

func NewUsbDevice(portn string, l *logger.Logger) (*UsbDevice, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		l.Error(err.Error())
		return nil, err
	}
	if len(ports) == 0 {
		l.Error("no ports has been found")
		return nil, err
	}
	for _, port := range ports {
		if port.IsUSB {
			vid, err := strconv.ParseUint(port.VID, 16, 64)
			if err != nil {
				return nil, fmt.Errorf("could not parse VendorId: %w", err)
			}
			pid, err := strconv.ParseUint(port.PID, 16, 64)
			if err != nil {
				return nil, fmt.Errorf("could not parse ProducId: %w", err)
			}
			if portn == "AUTO" && port.SerialNumber == "20080411" {
				return &UsbDevice{
					ProducId:     uint16(pid),
					VendorId:     uint16(vid),
					SerialNumber: port.SerialNumber,
					Name:         port.Name,
					log:          l,
				}, nil
			} else if portn == "AUTO" && port.SerialNumber == "USB7INCH" {
				l.Info("device is sleeping, let's wake ip up...(its lazy, 20 seconds to wake up!)")
				wakeUpDevice(port.Name, l)
				time.Sleep(20 * time.Second)
				l.Info("detecting again...")
				return NewUsbDevice(portn, l)
			} else if portn == port.Name {
				return &UsbDevice{
					ProducId:     uint16(pid),
					VendorId:     uint16(vid),
					SerialNumber: port.SerialNumber,
					Name:         port.Name,
					log:          l,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("no matching ports has been found")
}

func wakeUpDevice(name string, l *logger.Logger) {
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
		InitialStatusBits: &serial.ModemOutputBits{
			RTS: true,
			DTR: true,
		},
	}
	l.Infof("waking up device on: %s", name)
	port, err := serial.Open(name, mode)
	if err != nil {
		l.Errorf("could not open a device: %s", err)
	}
	port.Close()
}

func (u UsbDevice) ResetDevice() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	// Initialize a new Context.
	ctx := gousb.NewContext()
	defer ctx.Close()

	u.log.Info("reseting the device....")

	dev, err := ctx.OpenDeviceWithVIDPID(gousb.ID(u.VendorId), gousb.ID(u.ProducId))
	defer dev.Close()

	if err != nil {
		u.log.Errorf("could not open a device: %s", err)
		return err
	}
	err = dev.Reset()
	if err != nil {
		u.log.Errorf("could not reset device: %s", err)
		return err
	}
	return nil
}
