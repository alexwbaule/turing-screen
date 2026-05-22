package command

import (
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
)

type PreUpdateBitmap struct {
	bytes   []byte
	name    string
	padding byte
	log     *logger.Logger
}

func NewPreUpdateBitmap(log *logger.Logger) *PreUpdateBitmap {
	return &PreUpdateBitmap{
		name: "PRE_UPDATE_BITMAP",
		bytes: []byte{
			0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		log:     log,
	}
}

func (p *PreUpdateBitmap) GetBytes() [][]byte {
	tmp := utils.BZero(250, p.padding)
	copy(tmp, p.bytes)
	return [][]byte{tmp}
}

func (p *PreUpdateBitmap) SetCount(count int64) {
	_ = count
}

func (p *PreUpdateBitmap) GetName() string {
	return p.name
}

func (p *PreUpdateBitmap) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  0,
		Bytes: nil,
	}
}

func (p *PreUpdateBitmap) ValidateCommand([]byte, int) error {
	return nil
}
