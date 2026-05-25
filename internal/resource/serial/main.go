package serial

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/alexwbaule/serial"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/resource/usb"
)

const attempts = 3
const retransmitAttempts = 3
const retransmitDelay = 100 * time.Millisecond

// ReconnectConfig holds configuration for exponential backoff reconnection.
type ReconnectConfig struct {
	InitialDelay time.Duration // Starting delay between reconnection attempts (default: 1s)
	MaxDelay     time.Duration // Maximum delay between reconnection attempts (default: 60s)
	MaxAttempts  int           // Maximum number of consecutive reconnection attempts (default: 10)
}

// DefaultReconnectConfig returns the default reconnection configuration.
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
		MaxAttempts:  10,
	}
}

// SerialPort defines the interface for serial port operations used by the Serial driver.
// This enables testing with mock implementations.
type SerialPort interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Flush() error
	Close() error
}

type Serial struct {
	device          *usb.UsbDevice
	port            SerialPort
	log             *logger.Logger
	reconnectConfig ReconnectConfig
}

type SerialSender interface {
	Write(p command.Command) (int, error)
	WriteBytes(p command.Command) (int, error)
	WriteRaw(data []byte) (int, error)
	ReadPoll(maxWait time.Duration) ([]byte, error)
	Read(p command.Command) (int, error)
	RestartConnection() error
	ResetDevice() error
	Reconnect(ctx context.Context) error
}

func NewSerial(portName string, l *logger.Logger) (*Serial, error) {
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
		Size:        8,
		InitialDTR:  new(true),
		InitialRTS:  new(true),
	}
	port, err := serial.OpenPort(config)
	if err != nil {
		return nil, fmt.Errorf("error opening port %s: %w", device.Name, err)
	}

	return &Serial{
		device:          device,
		port:            port,
		log:             l,
		reconnectConfig: DefaultReconnectConfig(),
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
func (s *Serial) RestartConnection() error {
	s.log.Info("Restarting serial port connection")
	err := s.Close()
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
	s.log.Info("Restarting device")
	err := s.Close()
	if err != nil {
		return err
	}
	/*
		// TESTING IF IS REALLY NECESSARY DO THIS.
			err = s.device.ResetDevice()
			if err != nil {
				return err
			}
	*/
	err = s.ReopenPort()
	if err != nil {
		return err
	}
	return nil
}

// Reconnect closes the current port and attempts to reopen the serial connection
// using exponential backoff. The delay between attempts follows the formula:
// delay = min(InitialDelay × 2^(n-1), MaxDelay) where n is the attempt number (1-based).
// On success it returns nil and the caller is responsible for reinitializing the device.
// After MaxAttempts consecutive failures, it logs a fatal error and returns an error.
// The method respects context cancellation for graceful shutdown.
func (s *Serial) Reconnect(ctx context.Context) error {
	cfg := s.reconnectConfig
	s.log.Infof("Starting reconnection sequence (max %d attempts)", cfg.MaxAttempts)

	// Close the current port (best-effort, ignore errors since port may already be broken)
	_ = s.port.Close()

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Calculate exponential backoff delay: InitialDelay × 2^(attempt-1), capped at MaxDelay
		delay := cfg.InitialDelay * (1 << (attempt - 1))
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}

		s.log.Infof("Reconnection attempt %d/%d, waiting %v", attempt, cfg.MaxAttempts, delay)

		// Wait for the backoff delay, respecting context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("reconnection cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}

		// Attempt to reopen the port
		v, err := NewSerial(s.device.Name, s.log)
		if err != nil {
			s.log.Warnf("Reconnection attempt %d/%d failed: %v", attempt, cfg.MaxAttempts, err)
			continue
		}

		// Success: update the port and device references
		s.port = v.port
		s.device = v.device
		s.log.Infof("Reconnection successful on attempt %d", attempt)
		return nil
	}

	s.log.Errorf("Reconnection failed after %d consecutive attempts, service will terminate", cfg.MaxAttempts)
	return fmt.Errorf("reconnection failed: exhausted %d attempts", cfg.MaxAttempts)
}

func (s *Serial) Close() error {
	s.log.Info("Closing serial port..")
	err := s.port.Flush()
	if err != nil {
		return fmt.Errorf("serial flush error: %w", err)
	}
	s.log.Info("Done!")
	return s.port.Close()
}

// WriteRaw writes raw bytes directly to the serial port without any framing
// or padding. Used for file upload data after CREATE_FILE.
func (s *Serial) WriteRaw(data []byte) (int, error) {
	n, err := s.port.Write(data)
	if err != nil {
		return n, fmt.Errorf("write raw error: %w", err)
	}
	return n, nil
}

// ReadPoll polls the serial port for a response with 1-second timeout intervals.
// Returns the first non-empty response, or error after maxWait duration.
// Used to wait for "file_rev_done" after upload (device takes 5-15s to write flash).
func (s *Serial) ReadPoll(maxWait time.Duration) ([]byte, error) {
	deadline := time.Now().Add(maxWait)
	buf := make([]byte, 1024)

	for time.Now().Before(deadline) {
		n, err := s.port.Read(buf)
		if n > 0 {
			return buf[:n], nil
		}
		if err != nil && err != io.EOF {
			// Timeout is expected, continue polling
			continue
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("read poll timeout after %v", maxWait)
}

func (s *Serial) Write(p command.Command) (int, error) {
	var writen int
	for _, b := range p.GetBytes() {
		n, err := s.port.Write(b)
		writen += n
		if n == 0 {
			err = s.writeBackoff(b)
		}
		if err != nil {
			return 0, fmt.Errorf("write serial error: %w", err)
		}
	}
	v := p.ValidateWrite()
	if v.Bytes != nil {
		n, err := s.port.Write(v.Bytes)
		writen += n
		if n == 0 {
			err = s.writeBackoff(v.Bytes)
		}
		if err != nil {
			return 0, fmt.Errorf("write serial error: %w", err)
		}
	}
	if v.Size > 0 {
		return s.Read(p)
	}
	return writen, nil
}

// WriteBytes writes only the command byte chunks to the serial port without
// sending QUERY_STATUS or reading a response. Used for batch processing where
// multiple UPDATE_BITMAP commands are sent before a single QUERY_STATUS.
func (s *Serial) WriteBytes(p command.Command) (int, error) {
	var writen int
	for _, b := range p.GetBytes() {
		n, err := s.port.Write(b)
		writen += n
		if n == 0 {
			err = s.writeBackoff(b)
		}
		if err != nil {
			return 0, fmt.Errorf("write serial error: %w", err)
		}
	}
	return writen, nil
}

func (s *Serial) Read(p command.Command) (int, error) {
	var readed int
	var trying = 0

	v := p.ValidateWrite()
	buff := make([]byte, v.Size)

	for {
		// Read into the remaining space of buff
		n, err := s.port.Read(buff[readed:])
		readed += n
		trying++

		if err != nil && err != io.EOF {
			return 0, fmt.Errorf("read serial error: %w", err)
		}

		if readed > 0 {
			response := string(bytes.Trim(buff[:readed], "\x00"))

			if strings.Contains(response, "needReSend:1") {
				s.log.Warnf("Device requested retransmission: %s", response)
				return readed, s.handleNeedReSend(p)
			}
			if strings.Contains(response, "needReSend:0") {
				s.log.Debugf("Device confirmed receipt: %s", response)
				return readed, nil
			}

			// Validate with the command logic
			if valErr := p.ValidateCommand(buff[:readed], readed); valErr == nil {
				return readed, nil
			}
		}

		if n == 0 && trying <= attempts {
			s.log.Warnf("Readed zero, trying again [%d]", trying)
			continue
		}

		if readed == 0 {
			return 0, fmt.Errorf("read serial error: no response")
		}

		if readed >= v.Size {
			break
		}
		if n == 0 && trying > attempts {
			break
		}
	}

	// Final validation check if we break out of the loop
	err := p.ValidateCommand(buff[:readed], readed)
	if err != nil {
		s.log.Debugf("Error on validate, readed [%s] = %s", string(bytes.Trim(buff[:readed], "\x00")), err.Error())
		return readed, err // Return the read bytes along with the error
	}
	return readed, nil
}

func (s *Serial) writeBackoff(b []byte) error {
	delays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}

	for i := 0; i < attempts; i++ {
		time.Sleep(delays[i])
		n, err := s.port.Write(b)
		if n > 0 {
			return nil
		}
		if err != nil {
			s.log.Warnf("writeBackoff attempt %d failed: %v", i+1, err)
		} else {
			s.log.Warnf("writeBackoff attempt %d: zero bytes written", i+1)
		}
	}
	return fmt.Errorf("write serial error: all %d retry attempts returned zero bytes written", attempts)
}

// handleNeedReSend retransmits all byte chunks of the last command up to 3 times
// with 100ms between attempts. It flushes the serial port read buffer before each
// response read during retransmission. Returns nil on success (needReSend:0),
// or an error after 3 failed retransmission attempts.
func (s *Serial) handleNeedReSend(lastCommand command.Command) error {
	v := lastCommand.ValidateWrite()

	for attempt := 1; attempt <= retransmitAttempts; attempt++ {
		s.log.Warnf("Retransmitting command [%s], attempt %d/%d", lastCommand.GetName(), attempt, retransmitAttempts)

		time.Sleep(retransmitDelay)

		// Retransmit all byte chunks of the command
		for _, b := range lastCommand.GetBytes() {
			n, err := s.port.Write(b)
			if err != nil {
				return fmt.Errorf("retransmit write error on attempt %d: %w", attempt, err)
			}
			if n == 0 {
				return fmt.Errorf("retransmit write error on attempt %d: zero bytes written", attempt)
			}
		}

		// Write the validation bytes if present (e.g., QueryStatus)
		if v.Bytes != nil {
			n, err := s.port.Write(v.Bytes)
			if err != nil {
				return fmt.Errorf("retransmit validation write error on attempt %d: %w", attempt, err)
			}
			if n == 0 {
				return fmt.Errorf("retransmit validation write error on attempt %d: zero bytes written", attempt)
			}
		}

		// Flush the read buffer before reading the response (Req 4.5)
		err := s.port.Flush()
		if err != nil {
			return fmt.Errorf("retransmit flush error on attempt %d: %w", attempt, err)
		}

		// Read the response using the same logic as Read()
		buff := make([]byte, v.Size)
		var readed int
		var trying int
		success := false

		for {
			n, err := s.port.Read(buff[readed:])
			readed += n
			trying++

			if err != nil && err != io.EOF {
				return fmt.Errorf("retransmit read error on attempt %d: %w", attempt, err)
			}

			if readed > 0 {
				response := string(bytes.Trim(buff[:readed], "\x00"))

				if strings.Contains(response, "needReSend:0") {
					s.log.Infof("Retransmit successful on attempt %d: %s", attempt, response)
					return nil
				}
				if strings.Contains(response, "needReSend:1") {
					s.log.Warnf("Retransmit attempt %d: device still requesting resend: %s", attempt, response)
					break // break inner loop to retry retransmission attempt
				}
				if valErr := lastCommand.ValidateCommand(buff[:readed], readed); valErr == nil {
					s.log.Infof("Retransmit successful (validated) on attempt %d", attempt)
					return nil
				}
			}

			if n == 0 && trying <= attempts {
				continue
			}
			if readed >= v.Size || (n == 0 && trying > attempts) {
				break
			}
		}

		if !success {
			s.log.Warnf("Retransmit attempt %d failed to get valid confirmation", attempt)
		}
	}

	return fmt.Errorf("retransmission failed after %d attempts for command [%s]", retransmitAttempts, lastCommand.GetName())
}
