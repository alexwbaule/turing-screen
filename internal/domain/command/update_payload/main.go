package update_payload

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"math/big"
	"regexp"
	"time"
)

const chunk = 249

type UpdatePayload struct {
	bytes   []byte
	payload []byte
	name    string
	padding byte
	size    int
	count   int
	readed  *regexp.Regexp
	log     *logger.Logger
	start   time.Time
}

func NewUpdatePayload(log *logger.Logger) *UpdatePayload {
	return &UpdatePayload{
		name: "UPDATE_BITMAP",
		bytes: []byte{
			0xcc, 0xef, 0x69, 0x00, 0x00,
		},
		padding: 0x00,
		count:   -1,
		log:     log,
		start:   time.Now(),
	}
}

func (m *UpdatePayload) GetBytes() [][]byte {
	var fullImage [][]byte
	var updateBitMapCmd = utils.BZero(250, m.padding)

	size := len(m.payload)
	pSize := utils.PadBegin(big.NewInt(int64(size)).Bytes(), 2)
	pCount := utils.PadBegin(big.NewInt(int64(m.count)).Bytes(), 4)
	pPad := make([]byte, 3)

	copy(updateBitMapCmd, m.bytes)
	copy(updateBitMapCmd[5:], pSize)
	copy(updateBitMapCmd[7:], pPad)
	copy(updateBitMapCmd[10:], pCount)

	m.log.Debugf("Count: %d", m.count)

	//m.log.Infof("[%v]\n", hex.EncodeToString(updateBitMapCmd))

	fullImage = append(fullImage, updateBitMapCmd)

	for i := 0; i < size; i += chunk {
		end := i + chunk
		if end > size {
			end = size
		}
		tmp := utils.BZero(250, m.padding)
		copy(tmp, m.payload[i:end])
		fullImage = append(fullImage, tmp)
	}
	return fullImage
}

func (m *UpdatePayload) GetName() string {
	return m.name
}

func (m *UpdatePayload) GetSize() int {
	return m.size
}

func (m *UpdatePayload) ValidateCommand(s []byte, i int) error {
	return nil
}

func (m *UpdatePayload) GetFPS() string {
	t := time.Since(m.start)
	v := float64(m.count) / t.Seconds()
	return fmt.Sprintf("Count: %d Time: %s FPS: %.2f", m.count, t, v)
}

func (m *UpdatePayload) SendPayload(partial device.ImagePartial, x, y int) *UpdatePayload {
	m.payload = partial.GeneratePartialImage(x, y)
	m.count++
	return m
}
