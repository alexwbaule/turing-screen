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
		for {
			select {
			case <-w.ctx.Done():
				w.log.Infof("Stopping worker job...")
				return context.Canceled
			case item := <-jobs:
				w.log.Infof("worker job: %d", len(jobs))

				item.SetCount(num)

				switch item.(type) {
				case *command.UpdatePayload:
					num++
				}
				_, err := w.sender.Write(item)
				//w.log.Debugf("WorkerRun writed: %d", write)
				if err != nil {
					if try == attempts {
						w.log.Errorf("worker error: %s", err.Error())
						return err
					}
					w.log.Errorf("retry [%d] worker on error: %s", try, err.Error())
					err := fnConnError()
					if err != nil {
						return err
					}
					try++
					num = 0
					continue
				}
			}
		}
	}
}
