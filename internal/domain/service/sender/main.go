package sender

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command/brightness"
	"github.com/alexwbaule/turing-screen/internal/domain/command/device"
	"github.com/alexwbaule/turing-screen/internal/domain/command/media"
	"github.com/alexwbaule/turing-screen/internal/domain/command/option"
	"github.com/alexwbaule/turing-screen/internal/domain/command/payload"
	"github.com/alexwbaule/turing-screen/internal/domain/command/update_payload"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
)

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

func (w *Worker) Run(jobs <-chan any) error {
	var err error
	for {
		select {
		case <-w.ctx.Done():
			w.log.Infof("Stopping worker job...")
			return context.Canceled
		case item := <-jobs:
			switch item.(type) {
			case *brightness.Brightness:
				cmd := item.(*brightness.Brightness)
				_, err = w.sender.Write(cmd)
			case *device.Device:
				cmd := item.(*device.Device)
				_, err = w.sender.Write(cmd)
			case *media.Media:
				cmd := item.(*media.Media)
				_, err = w.sender.Write(cmd)
			case *option.Option:
				cmd := item.(*option.Option)
				_, err = w.sender.Write(cmd)
			case *payload.Payload:
				cmd := item.(*payload.Payload)
				_, err = w.sender.Write(cmd)
			case *update_payload.UpdatePayload:
				cmd := item.(*update_payload.UpdatePayload)
				_, err = w.sender.Write(cmd)
			}
			if err != nil {
				w.log.Errorf("worker error: %s", err.Error())
				return err
			}
		}
	}
}
