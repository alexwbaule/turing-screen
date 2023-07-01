package device_serial

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/command"
	"github.com/alexwbaule/turing-screen/usb"
	"go.bug.st/serial"
	"golang.org/x/exp/slog"
	"time"
)

type Serial struct {
	device *usb.UsbDevice
	port   serial.Port
	log    *slog.Logger
}

func NewSerial(device *usb.UsbDevice, l *slog.Logger) (*Serial, error) {
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
	l.Info(fmt.Sprintf("Connecting Using: %s", device.Name))
	port, err := serial.Open(device.Name, mode)
	if err != nil {
		return nil, err
	}
	err = port.SetReadTimeout(5 * time.Second)
	if err != nil {
		return nil, err
	}
	return &Serial{
		device: device,
		port:   port,
		log:    l,
	}, nil
}

func (s *Serial) Write(p command.Command) (int, error) {
	n, err := s.port.Write(p.GetBytes())
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Serial) Read(p command.Command) (int, error) {
	var readed int
	buff := make([]byte, p.GetSize())
	for {
		n, err := s.port.Read(buff)
		readed += n
		s.log.Info(fmt.Sprintf("Readed %v bytes", readed))
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, fmt.Errorf("no response")
		}
		if n == p.GetSize() {
			break
		}
	}
	err := p.ValidateCommand(buff, readed)
	if err != nil {
		return 0, err
	}
	return readed, nil
}
