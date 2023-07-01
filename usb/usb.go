package usb

import (
	"fmt"
	"github.com/google/gousb"
	"go.bug.st/serial/enumerator"
	"golang.org/x/exp/slog"
	"strconv"
)

type UsbDevice struct {
	ProducId     uint16
	VendorId     uint16
	SerialNumber string
	Name         string
	log          *slog.Logger
}

func NewUsbDevice(portn string, l *slog.Logger) (*UsbDevice, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		l.Error(err.Error())
		return nil, err
	}
	if len(ports) == 0 {
		l.Error("No ports has been found")
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
			if portn == "AUTO" && (port.SerialNumber == "20080411" || port.SerialNumber == "USB7INCH") {
				return &UsbDevice{
					ProducId:     uint16(pid),
					VendorId:     uint16(vid),
					SerialNumber: port.SerialNumber,
					Name:         port.Name,
					log:          l,
				}, nil
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

func (u UsbDevice) ResetDevice() {
	// Initialize a new Context.
	ctx := gousb.NewContext()
	defer ctx.Close()

	u.log.Info("Reseting the device....")

	dev, err := ctx.OpenDeviceWithVIDPID(gousb.ID(u.VendorId), gousb.ID(u.ProducId))
	defer dev.Close()

	if err != nil {
		u.log.Error(fmt.Sprintf("Could not open a device: %s", err))
		return
	}
	err = dev.Reset()
	if err != nil {
		u.log.Error(fmt.Sprintf("Could not reset device: %s", err))
		return
	}
}
