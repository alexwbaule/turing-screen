package serial

import (
	"bytes"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/resource/usb"
	"go.bug.st/serial"
	"time"
)

type Serial struct {
	device *usb.UsbDevice
	port   serial.Port
	log    *logger.Logger
}

type SerialSender interface {
	Write(p command.Command) (int, error)
	Read(p command.Command) (int, error)
}

func NewSerial(portName string, l *logger.Logger) (*Serial, error) {
	device, err := usb.NewUsbDevice(portName, l)
	if err != nil {
		return nil, fmt.Errorf("error finding devices %s: %w", portName, err)
	}
	l.Infof("Connecting Using: %s", device.Name)

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

func (s *Serial) ReopenPort() error {
	s.log.Infof("Reopening serial port connection on %s", time.Second*10)

	time.Sleep(time.Second * 5)
	v, err := NewSerial(s.device.Name, s.log)
	if err != nil {
		return err
	}
	s.port = v.port
	s.device = v.device
	return nil
}

func (s *Serial) RestartDevice() error {
	s.log.Info("Restarting serial port connection")
	err := s.Close()
	if err != nil {
		return err
	}
	err = s.ResetDevice()
	if err != nil {
		return err
	}
	err = s.ReopenPort()
	if err != nil {
		return err
	}
	return nil
}

func (s *Serial) ResetDevice() error {
	return s.device.ResetDevice()
}

func (s *Serial) Close() error {
	s.log.Info("Closing serial port")
	err := s.port.ResetInputBuffer()
	if err != nil {
		return fmt.Errorf("serial reset input buffer error: %w", err)
	}
	err = s.port.ResetOutputBuffer()
	if err != nil {
		return fmt.Errorf("serial reset output buffer error: %w", err)
	}
	s.log.Info("Closed serial port")
	return s.port.Close()
}

func (s *Serial) Write(p command.Command) (int, error) {
	var writen int
	//s.log.Debugf("Running Command %s", p.GetName())
	for _, b := range p.GetBytes() {
		n, err := s.port.Write(b)
		writen += n
		if err != nil {
			return 0, fmt.Errorf("write serial error: %w", err)
		}
	}
	//s.log.Debugf("Writen %d bytes", writen)
	v := p.ValidateWrite()
	if v.Bytes != nil {
		n, err := s.port.Write(v.Bytes)
		writen += n
		if err != nil {
			return 0, fmt.Errorf("write serial error: %w", err)
		}
	}
	if v.Size > 0 {
		return s.Read(p)
	}
	return writen, nil
}

func (s *Serial) Read(p command.Command) (int, error) {
	var readed int

	v := p.ValidateWrite()

	buff := make([]byte, v.Size)
	for {
		n, err := s.port.Read(buff)
		readed += n
		if err != nil {
			return 0, fmt.Errorf("read serial error: %w", err)
		}
		if n == 0 {
			return 0, fmt.Errorf("read serial error: no response")
		}
		if n == v.Size {
			break
		}
	}
	//s.log.Debugf("Readed %d bytes [%s]", readed, string(bytes.Trim(buff, "\x00")))

	err := p.ValidateCommand(buff, readed)
	if err != nil {
		s.log.Debugf("Error on validate, readed [%s] = %s", string(bytes.Trim(buff, "\x00")), err.Error())
		return 0, err
	}
	return readed, nil
}
