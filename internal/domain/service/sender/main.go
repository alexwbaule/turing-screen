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

func (w *Worker) Run(jobs <-chan command.Command, fnConnError func() error) error {
	var try = 0
	var num int64 = 0
	for {
		for {
			select {
			case <-w.ctx.Done():
				w.log.Infof("Stopping worker job...")
				return context.Canceled
			case item := <-jobs:
				item.SetCount(num)

				switch item.(type) {
				case *command.UpdatePayload:
					num++
				}
				write, err := w.sender.Write(item)
				w.log.Debugf("WorkerRun writed: %d", write)
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
					return fnConnError()
				}
			}
		}
	}
}
