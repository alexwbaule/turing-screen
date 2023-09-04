package sender

import (
	"context"
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

func (w *Worker) Run(jobs <-chan command.Command, fnConnError func() error) error {
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
					return err
				}
				w.log.Errorf("retry [%d] worker error: %s", try+1, err.Error())
				err := fnConnError()
				if err != nil {
					return err
				}
				try++
				num = 0
			}
		}
	}
}
