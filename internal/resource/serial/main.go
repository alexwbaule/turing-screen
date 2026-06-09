package serial

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/alexwbaule/serial"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/resource/usb"
)

const attempts = 3

// SerialSender defines the interface for interacting with the serial device.
type SerialSender interface {
	// Execute performs a synchronous command: writes, waits for a response, validates it, and returns the response.
	Execute(cmd command.Command) ([]byte, error)

	// Write sends raw bytes to the serial port. Used for asynchronous operations by the worker.
	Write(data []byte) (int, error)

	// Read reads raw bytes from the serial port into the given buffer.
	Read(p []byte) (int, error)

	// Flush discards all unread bytes in the OS receive buffer.
	// Call after stopping workers to prevent stale responses from corrupting the next session.
	Flush() error

	// Close terminates the connection.
	Close() error

	// ResetDevice performs a hardware reset and reopens the connection.
	ResetDevice() error
}

// Serial implements the SerialSender interface.
type Serial struct {
	device     *usb.UsbDevice
	port       *serial.Port
	log        *logger.Logger
	configPort string // original value from config ("AUTO" or explicit path)
}

// NewSerial creates and opens a new serial port connection.
func NewSerial(portName string, l *logger.Logger) (SerialSender, error) {
	device, err := usb.NewUsbDevice(portName, l)
	if err != nil {
		return nil, fmt.Errorf("error finding devices %s: %w", portName, err)
	}
	l.Infof("Connecting Using: %s", device.Name)

	config := &serial.Config{
		Baud:        115200,
		Name:        device.Name,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 5 * time.Second,
	}
	port, err := serial.OpenPort(config)
	if err != nil {
		return nil, fmt.Errorf("error opening port %s: %w", device.Name, err)
	}

	return &Serial{
		device:     device,
		port:       port,
		log:        l,
		configPort: portName,
	}, nil
}

// Execute performs a synchronous command execution.
func (s *Serial) Execute(cmd command.Command) ([]byte, error) {
	if s.port == nil {
		return nil, fmt.Errorf("serial port is closed")
	}
	s.log.Debugf("executing sync command: %s", cmd.GetName())

	// 1. Write all command packets
	for _, b := range cmd.GetBytes() {
		if _, err := s.Write(b); err != nil {
			return nil, fmt.Errorf("sync write failed for %s: %w", cmd.GetName(), err)
		}
	}

	// 2. Check if the command expects a response
	validation := cmd.ValidateWrite()
	if validation.Size == 0 {
		s.log.Debug("oneway command, without response...")
		return nil, nil
	}

	// 3. Read the response
	responseBuf := make([]byte, validation.Size)
	n, err := s.Read(responseBuf)
	if err != nil {
		return nil, fmt.Errorf("sync read failed for %s: %w", cmd.GetName(), err)
	}

	response := responseBuf[:n]
	s.log.Debugf("Readed: %s", string(bytes.Trim(response, "\x00")))

	// 4. Validate the response
	if err := cmd.ValidateCommand(response, n); err != nil {
		s.log.Infof("Command %s failed: %v", cmd.GetName(), err)
		return nil, err
	}

	s.log.Debugf("Sync command %s successful, response: %s", cmd.GetName(), string(bytes.Trim(response, "\x00")))
	return response, nil
}

// Write sends raw bytes to the serial port with a retry mechanism.
func (s *Serial) Write(data []byte) (int, error) {
	if s.port == nil {
		return 0, fmt.Errorf("port is closed")
	}
	n, err := s.port.Write(data)
	if err != nil {
		return 0, fmt.Errorf("write serial error: %w", err)
	}
	if n == 0 {
		// Retry logic if zero bytes were written
		for i := 0; i < attempts; i++ {
			if s.port == nil {
				return 0, fmt.Errorf("port is closed")
			}
			n, err = s.port.Write(data)
			if err != nil {
				return 0, fmt.Errorf("write retry error: %w", err)
			}
			if n > 0 {
				return n, nil
			}
		}
		return 0, fmt.Errorf("write serial error: zero bytes written after %d attempts", attempts)
	}
	return n, nil
}

// Read reads from the serial port with retry logic for empty reads.
func (s *Serial) Read(p []byte) (int, error) {
	if s.port == nil {
		return 0, fmt.Errorf("port is closed")
	}

	for attempt := 0; attempt < attempts; attempt++ {
		n, err := s.port.Read(p)
		if n > 0 {
			// Data received — return it even if err == io.EOF.
			// CDC-ACM devices sometimes close the stream after the last response byte,
			// which produces (n>0, io.EOF). Discarding n here was causing DeleteFile
			// to lose "delete_success" and exhaust retries.
			return n, nil
		}
		if err != nil {
			if err == io.EOF {
				s.log.Warnf("Read EOF, trying again")
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return 0, fmt.Errorf("read serial error: %w", err)
		}
		// Zero bytes, no error — retry
		time.Sleep(100 * time.Millisecond)
	}

	return 0, fmt.Errorf("read serial error: no response after %d attempts", attempts)
}

func (s *Serial) ReopenPort() error {
	s.log.Infof("Reopening serial port connection on %s", s.device.Name)
	time.Sleep(time.Second * 5)

	newSerial, err := NewSerial(s.configPort, s.log)
	if err != nil {
		return err
	}

	// This is not an idiomatic way to re-assign. It's better to handle this at a higher level.
	// For now, let's just update the internal fields.
	if newConcrete, ok := newSerial.(*Serial); ok {
		s.port = newConcrete.port
		s.device = newConcrete.device
	} else {
		return fmt.Errorf("failed to cast new serial instance")
	}

	return nil
}

func (s *Serial) RestartConnection() error {
	s.log.Info("Restarting serial port connection")
	if err := s.Close(); err != nil {
		return err
	}
	return s.ReopenPort()
}

func (s *Serial) ResetDevice() error {
	s.log.Info("Resetting device")
	if err := s.Close(); err != nil {
		// Log the error but continue, as reopening is the main goal
		s.log.Warnf("error during close on reset: %v", err)
	}

	// The original code commented this out. Let's respect that.
	/*
		err = s.device.ResetDevice()
		if err != nil {
			return err
		}
	*/

	return s.ReopenPort()
}

// Flush drains all pending bytes from the device receive buffer.
// It sets a 100ms read timeout so each Read() returns immediately when data
// is present or after 100ms of silence when the buffer is empty. This catches
// bytes stuck in the USB subsystem layer that FIONREAD (BytesAvailable) misses,
// which was the root cause of serial contamination between storage commands.
func (s *Serial) Flush() error {
	if s.port == nil {
		return nil
	}
	if err := s.port.SetReadTimeout(100 * time.Millisecond); err != nil {
		// fallback: plain sleep if termios change fails
		time.Sleep(100 * time.Millisecond)
		return nil
	}
	defer s.port.SetReadTimeout(5 * time.Second)

	buf := make([]byte, 512)
	for {
		n, _ := s.port.Read(buf)
		if n == 0 {
			break // 100ms of silence — buffer is clear
		}
	}
	return nil
}

func (s *Serial) Close() error {
	s.log.Info("Closing serial port...")
	if s.port == nil {
		return nil
	}
	if err := s.port.Flush(); err != nil {
		s.log.Warnf("serial flush error on close: %v", err)
		// Don't return here, still try to close
	}
	err := s.port.Close()
	s.port = nil
	s.log.Info("Serial port closed.")
	return err
}
