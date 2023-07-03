package update_payload

import (
	"encoding/hex"
	"fmt"
	"github.com/alexwbaule/turing-screen/image_process"
	"github.com/alexwbaule/turing-screen/utils"
	"math/big"
	"regexp"
)

const chunk = 249

type UpdatePayload struct {
	bytes   []byte
	payload []byte
	name    string
	padding byte
	size    int
	count   int
	move    int
	readed  *regexp.Regexp
}

func NewUpdatePayload() *UpdatePayload {
	return &UpdatePayload{
		name: "UPDATE_BITMAP",
		bytes: []byte{
			0xcc, 0xef, 0x69, 0x00, 0x00,
		},
		padding: 0x00,
		count:   -1,
		move:    0,
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

	fmt.Printf("[%v]\n", hex.EncodeToString(updateBitMapCmd))

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

func (m *UpdatePayload) SendPayload(imagePath string) *UpdatePayload {
	img := image_process.NewImageProcess(image_process.LoadImage(imagePath))
	m.payload = img.GeneratePartialImage(m.move, 50)
	m.count++
	if m.move == 660 {
		m.move = 0
	} else {
		m.move++
	}
	return m
}
