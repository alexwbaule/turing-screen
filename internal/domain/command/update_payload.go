package command

import (
	"bytes"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	tdevice "github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"

	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"math/big"
	"regexp"
)

var (
	render = regexp.MustCompile(`^needReSend:0\|renderCnt:0$`)
)

type UpdatePayload struct {
	bytes       []byte
	payload     []byte
	name        string
	padding     byte
	size        int
	count       int64
	readed      *regexp.Regexp
	log         *logger.Logger
	orientation theme.Orientation
	device      *tdevice.Display
}

func NewUpdatePayload(log *logger.Logger, o theme.Orientation, t *tdevice.Display) *UpdatePayload {
	log.Infof("NewUpdatePayload: %d", 0)
	return &UpdatePayload{
		log:         log,
		orientation: o,
		device:      t,
	}
}

func (m *UpdatePayload) GetBytes() [][]byte {
	var fullImage [][]byte
	var updateBitMapCmd = utils.BZero(250, m.padding)

	size := len(m.payload)
	pSize := utils.PadBegin(big.NewInt(int64(size)).Bytes(), 2)
	pCount := utils.PadBegin(big.NewInt(m.count).Bytes(), 4)
	pPad := make([]byte, 3)

	copy(updateBitMapCmd, m.bytes)
	copy(updateBitMapCmd[5:], pSize)
	copy(updateBitMapCmd[7:], pPad)
	copy(updateBitMapCmd[10:], pCount)

	m.log.Debugf("Count: %d", m.count)

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

func (m *UpdatePayload) SetCount(count int64) {
	m.count = count
}

func (m *UpdatePayload) GetName() string {
	return m.name
}

func (m *UpdatePayload) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  m.size,
		Bytes: m.QueryStatus(),
	}
}
func (m *UpdatePayload) QueryStatus() []byte {
	tmp := utils.BZero(250, 0x00)
	copy(tmp, []byte{0xcf, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01})
	return tmp
}

func (m *UpdatePayload) ValidateCommand(s []byte, i int) error {
	v := string(bytes.Trim(s, "\x00"))
	if i == m.size && m.readed.MatchString(v) {
		return nil
	}
	return ErrMatch
}

func (m *UpdatePayload) SendPayload(partial device.ImagePartial, x, y int) *UpdatePayload {

	return &UpdatePayload{
		name: "UPDATE_BITMAP",
		bytes: []byte{
			0xcc, 0xef, 0x69, 0x00, 0x00,
		},
		padding: 0x00,
		size:    1024,
		readed:  render,
		payload: partial.GeneratePartialImage(m.orientation, m.device, x, y),
		log:     m.log,
	}
}
