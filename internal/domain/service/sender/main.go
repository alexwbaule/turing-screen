package sender

import (
	"context"
	"fmt"
	"time"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
)

const attempts = 3

type Worker struct {
	sender  serial.SerialSender
	log     *logger.Logger
	bg      device.ImageBackground
	device  *command.Device
	media   *command.Media
	payload *command.Payload
}

func NewWorker(s serial.SerialSender, background device.ImageBackground,
	d *command.Device, m *command.Media, p *command.Payload, l *logger.Logger) *Worker {
	return &Worker{
		sender:  s,
		bg:      background,
		log:     l,
		device:  d,
		media:   m,
		payload: p,
	}
}

func (w *Worker) Run(ctx context.Context, jobs <-chan command.Command) error {
	var try = 0
	var num int64 = 0

	for {
		select {
		case <-ctx.Done():
			w.log.Infof("stopping worker with %d updates.", num)
			return ctx.Err()
		case item := <-jobs:
			w.log.Debugf("queue size: %d - update payload num: %d", len(jobs), num)

			item.SetCount(num)

			switch item.(type) {
			case *command.UpdatePayload:
				num++
			}
			err := w.OffChannel(item)
			if err != nil {
				if try == attempts {
					w.log.Errorf("update %d, max attempts reached: %s", num, err.Error())
					return err
				}
				w.log.Errorf("update %d, retry [%d] worker error: %s", num, try+1, err.Error())
				err = w.OffChannel(w.device.Restart())
				if err != nil {
					_ = w.sender.ResetDevice()
				}
				try++
				num = 0
			} else {
				try = 0
			}
		}
	}
}

func (w *Worker) OffChannel(cmd command.Command) error {
	w.log.Debugf("worker processing command: %s", cmd.GetName())
	startTime := time.Now()

	// 1. Write all command packets
	for _, bytesToWrite := range cmd.GetBytes() {
		if _, err := w.sender.Write(bytesToWrite); err != nil {
			return fmt.Errorf("worker write failed for %s: %w", cmd.GetName(), err)
		}
	}

	// 2. Check if a response is expected
	validation := cmd.ValidateWrite()
	if validation.Size == 0 {
		w.log.Debugf("command %s sent successfully (no response expected)", cmd.GetName())
		return nil // Command sent, no response to read
	}

	// 2.5. Send QueryStatus bytes if required (triggers device response)
	if validation.Bytes != nil {
		if _, err := w.sender.Write(validation.Bytes); err != nil {
			return fmt.Errorf("worker query status write failed for %s: %w", cmd.GetName(), err)
		}
	}

	// 3. If necessary, read the response
	responseBuf := make([]byte, validation.Size)
	n, err := w.sender.Read(responseBuf)
	if err != nil {
		return fmt.Errorf("worker read failed for %s: %w", cmd.GetName(), err)
	}

	// 4. Validate the response
	if err := cmd.ValidateCommand(responseBuf, n); err != nil {
		return fmt.Errorf("invalid response for worker command %s: %w", cmd.GetName(), err)
	}

	w.log.Debugf("command %s processed successfully in %s", cmd.GetName(), time.Since(startTime))
	return nil
}
