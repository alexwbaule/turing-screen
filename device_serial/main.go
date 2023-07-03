package device_serial

import (
	"bytes"
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

func NewSerial(portName string, l *slog.Logger) (*Serial, error) {
	device, err := usb.NewUsbDevice(portName, l)
	if err != nil {
		return nil, fmt.Errorf("error finding devices %s: %w", portName, err)
	}
	l.Info(fmt.Sprintf("Connecting Using: %s", device.Name))

	config := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
		InitialStatusBits: &serial.ModemOutputBits{
			RTS: true,
			DTR: true,
		},
	}
	port, err := serial.Open(device.Name, config)
	if err != nil {
		return nil, fmt.Errorf("error opening port %s: %w", device.Name, err)
	}
	err = port.SetReadTimeout(5 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("error setting read timeout %s: %w", device.Name, err)
	}
	return &Serial{
		device: device,
		port:   port,
		log:    l,
	}, nil
}

func (s *Serial) ResetDevice() error {
	return s.device.ResetDevice()
}

func (s *Serial) Write(p command.Command) (int, error) {
	var writen int
	for _, b := range p.GetBytes() {
		//if p.GetName() == "UPDATE_BITMAP" {
		//s.log.Info(fmt.Sprintf("cmd: %s [%v]\n", p.GetName(), hex.EncodeToString(b)))
		//}
		n, err := s.port.Write(b)
		writen += n
		if err != nil {
			s.log.Info(fmt.Sprintf("error: %d (%s)\n", writen, err))
			return 0, err
		}
	}
	if p.GetSize() > 0 {
		return s.Read(p)
	}
	return writen, nil
}

func (s *Serial) Read(p command.Command) (int, error) {
	var readed int
	buff := make([]byte, p.GetSize())
	for {
		n, err := s.port.Read(buff)
		readed += n
		s.log.Info(fmt.Sprintf("Readed %v bytes", readed))
		s.log.Info(fmt.Sprintf("Readed %s", string(bytes.Trim(buff, "\x00"))))

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
