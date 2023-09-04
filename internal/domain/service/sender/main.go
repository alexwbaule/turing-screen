package sender

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
)

const attempts = 3

type Worker struct {
	sender  serial.SerialSender
	log     *logger.Logger
	ctx     context.Context
	bg      device.ImageBackground
	device  *command.Device
	media   *command.Media
	payload *command.Payload
}

func NewWorker(c context.Context, s serial.SerialSender, background device.ImageBackground,
	d *command.Device, m *command.Media, p *command.Payload, l *logger.Logger) *Worker {
	return &Worker{
		ctx:     c,
		sender:  s,
		bg:      background,
		log:     l,
		device:  d,
		media:   m,
		payload: p,
	}
}

func (w *Worker) Run(jobs <-chan command.Command) error {
	var try = 0
	var num int64 = 0

	for {
		select {
		case <-w.ctx.Done():
			w.log.Infof("Stopping worker")
			return w.ctx.Err()
		case item := <-jobs:
			w.log.Debugf("queue size: %d - update payload num: %d", len(jobs), num)

			item.SetCount(num)

			switch item.(type) {
			case *command.UpdatePayload:
				num++
			}
			_, err := w.sender.Write(item)
			if err != nil {
				if try == attempts {
					w.log.Errorf("worker error: %s", err.Error())
					_ = w.OffChannel(w.device.TurnOff())
					return err
				}
				w.log.Errorf("retry [%d] worker error: %s", try+1, err.Error())
				err := w.backoff()
				if err != nil {
					return err
				}
				try++
				num = 0
			}
		}
	}
}

func (w *Worker) backoff() error {
	err := w.OffChannel(w.media.StopVideo())
	if err != nil {
		return err
	}
	err = w.OffChannel(w.media.StopMedia())
	if err != nil {
		return err
	}
	err = w.OffChannel(w.device.Hello())
	if err != nil {
		return err
	}
	err = w.OffChannel(w.media.StopVideo())
	if err != nil {
		return err
	}
	err = w.OffChannel(w.media.StopMedia())
	if err != nil {
		return err
	}
	err = w.OffChannel(w.payload.SendPayload(w.bg))
	if err != nil {
		return err
	}
	return nil
}

func (w *Worker) OffChannel(cmd command.Command) error {
	write, err := w.sender.Write(cmd)
	if err != nil {
		w.log.Errorf("can't send command to device, bytes [%d] -> %s", write, err)
	}
	return err
}
