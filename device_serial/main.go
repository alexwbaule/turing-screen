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

func (s *Serial) Write(p []byte) (int, error) {
	n, err := s.port.Write(p)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Serial) Read(p *[]byte, i int) (int, error) {
	var readed int
	for {
		n, err := s.port.Read(*p)
		readed += n
		s.log.Info(fmt.Sprintf("Readed %v bytes", readed))
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, fmt.Errorf("no response")
		}
		if n == i {
			break
		}
	}
	return readed, nil
}

func (s *Serial) SendHello() error {
	cmd := command.NewCommand()
	var readed int
	var writed int

	writed, err := s.port.Write(cmd.Hello().GetBytes())
	if err != nil {
		return err
	}

	s.log.Info(fmt.Sprintf("Sent %v bytes", writed))

	buff := make([]byte, 23)
	for {
		n, err := s.port.Read(buff)
		readed += n
		s.log.Info(fmt.Sprintf("Readed %v bytes", readed))
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("no response")
		}
		if n == 23 {
			break
		}
	}
	if !cmd.Hello().ValidateCommand(buff[:23], readed) {
		return fmt.Errorf("no matching device")
	}
	s.log.Info("Matching Device!")
	return nil
}

func (s *Serial) SendStopMedia() error {
	var readed int
	var writed int

	cmd := command.NewCommand()

	cmd.StartDisplayBitmap().GetBytes()

	writed, err := s.port.Write(cmd.StopVideo().GetBytes())
	if err != nil {
		return err
	}
	s.log.Info(fmt.Sprintf("Sent %v bytes", writed))

	writed, err = s.port.Write(cmd.StopMedia().GetBytes())
	if err != nil {
		return err
	}
	s.log.Info(fmt.Sprintf("Sent %v bytes", writed))

	buff := make([]byte, 1024)
	for {
		n, err := s.port.Read(buff)
		readed += n
		s.log.Info(fmt.Sprintf("Readed %v bytes", readed))

		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("no response")
		}
		if n == 1024 {
			break
		}
	}
	if !cmd.StopMedia().ValidateCommand(buff[:10], readed) {
		return fmt.Errorf("no matching command")
	}
	s.log.Info("Matching Command!")
	return nil
}
