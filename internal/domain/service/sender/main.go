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

const maxBatchSize = 10

// healthCheckInterval is the duration of inactivity before a health check is sent.
const healthCheckInterval = 30 * time.Second

// maxHealthCheckFailures is the number of consecutive health check failures
// before calling ResetDevice.
const maxHealthCheckFailures = 3

type Worker struct {
	sender         serial.SerialSender
	log            *logger.Logger
	ctx            context.Context
	bg             device.ImageBackground
	device         *command.Device
	media          *command.Media
	payload        *command.Payload
	preUpdate      *command.PreUpdateBitmap
	queue          *RegionQueue
	healthCheck    *command.HealthCheck
	healthTick     *time.Ticker
	healthFailures int
	transmitting   bool
}

func NewWorker(c context.Context, s serial.SerialSender, background device.ImageBackground,
	d *command.Device, m *command.Media, p *command.Payload, pre *command.PreUpdateBitmap, h *command.HealthCheck, l *logger.Logger, q *RegionQueue) *Worker {
	return &Worker{
		ctx:         c,
		sender:      s,
		bg:          background,
		log:         l,
		device:      d,
		media:       m,
		payload:     p,
		preUpdate:   pre,
		healthCheck: h,
		queue:       q,
		healthTick:  time.NewTicker(healthCheckInterval),
	}
}

func (w *Worker) Run() error {
	var num int64 = 0

	defer w.healthTick.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.log.Infof("stopping worker with %d updates.", num)
			return w.ctx.Err()

		case <-w.queue.Notify():
			// Reset health tick since we're processing commands
			w.healthTick.Reset(healthCheckInterval)

			// Drain available commands in batches
			for {
				processed, n, err := w.processBatch(num)
				num += n
				if err != nil {
					// If context is already cancelled (shutdown), don't attempt reconnection
					if w.ctx.Err() != nil {
						return w.ctx.Err()
					}

					w.log.Errorf("update %d, worker error: %s", num, err.Error())

					// Start draining the queue in a goroutine to prevent sensor goroutines from blocking
					drainCtx, drainCancel := context.WithCancel(w.ctx)
					go w.drainQueueDuringReconnect(drainCtx)

					// Attempt reconnection with exponential backoff
					reconnErr := w.sender.Reconnect(w.ctx)
					drainCancel() // Stop draining once reconnection completes (success or failure)

					if reconnErr != nil {
						w.log.Errorf("reconnection failed: %s", reconnErr.Error())
						return fmt.Errorf("fatal: reconnection failed: %w", reconnErr)
					}

					// Reconnection succeeded — reinitialize device
					w.log.Info("reconnection successful, reinitializing device")
					err = w.backoff()
					if err != nil {
						w.log.Errorf("device reinitialization after reconnect failed: %s", err.Error())
						return fmt.Errorf("fatal: device reinitialization failed: %w", err)
					}

					// Reset counter on success
					num = 0
					break
				}
				if processed == 0 {
					// No more commands available
					break
				}
			}

		case <-w.healthTick.C:
			// Only send health check when not transmitting a command
			if w.transmitting {
				continue
			}
			err := w.performHealthCheck()
			if err != nil {
				w.healthFailures++
				w.log.Errorf("health check failed (%d/%d): %s", w.healthFailures, maxHealthCheckFailures, err.Error())

				if w.healthFailures >= maxHealthCheckFailures {
					w.log.Errorf("device unreachable after %d consecutive health check failures, resetting device", maxHealthCheckFailures)
					_ = w.sender.ResetDevice()
					w.healthFailures = 0
				} else {
					// Initiate reconnection via RestartConnection
					reconnErr := w.sender.RestartConnection()
					if reconnErr != nil {
						w.log.Errorf("reconnection after health check failure failed: %s", reconnErr.Error())
					}
				}
			} else {
				// Reset failure counter on successful health check
				w.healthFailures = 0
			}
		}
	}
}

// performHealthCheck sends a QUERY_STATUS command as a health probe and
// validates the device response. Returns nil on success, error on timeout
// or invalid response.
func (w *Worker) performHealthCheck() error {
	w.log.Debug("performing periodic health check")
	cmd := w.healthCheck.QueryStatus()
	_, err := w.sender.Write(cmd)
	if err != nil {
		return err
	}
	return nil
}

// processBatch dequeues up to 10 consecutive UPDATE_BITMAP commands, sends them all,
// then sends a single QUERY_STATUS. If a non-UPDATE_BITMAP command is encountered,
// it is sent immediately and the batch ends. Returns the number of commands processed,
// the count of UPDATE_BITMAP commands sent, and any error.
func (w *Worker) processBatch(startNum int64) (int, int64, error) {
	var bitmapBatch []command.Command
	var processed int
	var bitmapCount int64

	w.transmitting = true
	defer func() { w.transmitting = false }()

	for processed < maxBatchSize {
		cmd, ok := w.queue.Dequeue()
		if !ok {
			break
		}
		processed++

		// Non-UPDATE_BITMAP commands are sent immediately, ending the batch
		if _, isRegion := cmd.(command.RegionIdentifier); !isRegion {
			// First, flush any accumulated bitmap batch
			if len(bitmapBatch) > 0 {
				err := w.sendBitmapBatch(bitmapBatch, startNum+bitmapCount-int64(len(bitmapBatch)))
				if err != nil {
					return processed, bitmapCount, err
				}
				bitmapBatch = nil
			}
			// Send the non-bitmap command directly
			cmd.SetCount(startNum + bitmapCount)
			w.log.Debugf("queue size: %d - sending command: %s", w.queue.Len(), cmd.GetName())
			err := w.OffChannel(cmd)
			if err != nil {
				return processed, bitmapCount, err
			}
			return processed, bitmapCount, nil
		}

		// UPDATE_BITMAP command: accumulate in batch
		bitmapCount++
		cmd.SetCount(startNum + bitmapCount - 1)
		bitmapBatch = append(bitmapBatch, cmd)
	}

	// Send accumulated bitmap batch
	if len(bitmapBatch) > 0 {
		err := w.sendBitmapBatch(bitmapBatch, startNum)
		if err != nil {
			return processed, bitmapCount, err
		}
	}

	return processed, bitmapCount, nil
}

// sendBitmapBatch sends all UPDATE_BITMAP commands in the batch consecutively
// without sending QUERY_STATUS between each one, then sends a single QUERY_STATUS
// to validate the entire batch. If the device response after the batch indicates
// an error, it returns the error so the caller can initiate retry/backoff.
func (w *Worker) sendBitmapBatch(batch []command.Command, _ int64) error {
	w.log.Debugf("sending bitmap batch of %d commands, queue size: %d", len(batch), w.queue.Len())

	// Send all commands using WriteBytes (no QUERY_STATUS per command)
	for _, cmd := range batch {
		_, err := w.sender.WriteBytes(cmd)
		if err != nil {
			w.log.Errorf("batch command [%s] failed: %s", cmd.GetName(), err.Error())
			return err
		}
	}

	// Send a single QUERY_STATUS after the batch using the last command's validation
	lastCmd := batch[len(batch)-1]
	v := lastCmd.ValidateWrite()
	if v.Bytes != nil && v.Size > 0 {
		_, err := w.sender.Write(newBatchQueryCommand(lastCmd))
		if err != nil {
			w.log.Errorf("batch QUERY_STATUS failed: %s", err.Error())
			return err
		}
	}

	return nil
}

// drainQueueDuringReconnect continuously dequeues and discards commands from the
// queue while the serial driver is in reconnection state. This prevents sensor
// goroutines from blocking when the queue reaches capacity. The goroutine exits
// when the provided context is cancelled (i.e., reconnection completes or fails).
func (w *Worker) drainQueueDuringReconnect(ctx context.Context) {
	drained := 0
	for {
		select {
		case <-ctx.Done():
			if drained > 0 {
				w.log.Infof("drain during reconnect: discarded %d commands", drained)
			}
			return
		case <-w.queue.Notify():
			// Drain all available commands
			for {
				_, ok := w.queue.Dequeue()
				if !ok {
					break
				}
				drained++
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
	err = w.OffChannel(w.preUpdate)
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

// batchQueryCommand wraps the QUERY_STATUS portion of an UPDATE_BITMAP command
// so it can be sent via the SerialSender.Write interface after a batch of bitmap writes.
// It has no GetBytes (no image data to send), only the validation bytes (QUERY_STATUS).
type batchQueryCommand struct {
	source command.Command
}

func newBatchQueryCommand(source command.Command) *batchQueryCommand {
	return &batchQueryCommand{source: source}
}

func (b *batchQueryCommand) GetBytes() [][]byte {
	// No image bytes to send - only the QUERY_STATUS validation bytes
	return nil
}

func (b *batchQueryCommand) GetName() string {
	return "BATCH_QUERY_STATUS"
}

func (b *batchQueryCommand) ValidateCommand(data []byte, n int) error {
	return b.source.ValidateCommand(data, n)
}

func (b *batchQueryCommand) ValidateWrite() command.WriteValidation {
	return b.source.ValidateWrite()
}

func (b *batchQueryCommand) SetCount(num int64) {}
