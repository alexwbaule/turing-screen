package sender

import (
	"context"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/domain/command"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/resource/serial"
	"time"
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
			w.log.Infof("stopping worker with %d updates.", num)
			return w.ctx.Err()
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
				err = w.backoff()
				if err != nil {
					_ = w.sender.ResetDevice()
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
	now := time.Now()
	write, err := w.sender.Write(cmd)
	w.log.Debugf("time to write %s", time.Since(now))

	if err != nil {
		w.log.Errorf("can't send command [%s] to device, bytes [%d] -> %s", cmd.GetName(), write, err)
	}
	return err
}
