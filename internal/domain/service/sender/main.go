package sender

import (
	"context"
	"errors"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
)

const attempts = 3

type Worker struct {
	sender serial.SerialSender
	log    *logger.Logger
	ctx    context.Context
}

func NewWorker(c context.Context, s serial.SerialSender, l *logger.Logger) *Worker {
	return &Worker{
		ctx:    c,
		sender: s,
		log:    l,
	}
}

func (w *Worker) Run(jobs <-chan any, fn func() error) error {
	var err error
	var try = 0
	for {
		select {
		case <-w.ctx.Done():
			w.log.Infof("Stopping worker job...")
			return context.Canceled
		case item := <-jobs:
			switch item.(type) {
			case *command.Brightness:
				cmd := item.(*command.Brightness)
				_, err = w.sender.Write(cmd)
			case *command.Device:
				cmd := item.(*command.Device)
				_, err = w.sender.Write(cmd)
			case *command.Media:
				cmd := item.(*command.Media)
				_, err = w.sender.Write(cmd)
			case *command.Option:
				cmd := item.(*command.Option)
				_, err = w.sender.Write(cmd)
			case *command.Payload:
				cmd := item.(*command.Payload)
				_, err = w.sender.Write(cmd)
			case *command.UpdatePayload:
				cmd := item.(*command.UpdatePayload)
				_, err = w.sender.Write(cmd)
			}
			if err != nil {
				if errors.Is(err, command.ErrMatch) {
					w.log.Errorf("worker error: %s", err.Error())
					continue
				}
				if try == attempts {
					w.log.Errorf("worker error: %s", err.Error())
					return err
				}
				w.log.Errorf("retry [%d] worker on error: %s", try, err.Error())
				try++
				return fn()
			}
		}
	}
}
