package command

import (
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
)

type Brightness struct {
	name       string
	bytes      []byte
	brightness byte
	padding    byte
	log        *logger.Logger
}

func NewBrightness(log *logger.Logger) *Brightness {
	return &Brightness{
		name: "SetBrightness",
		bytes: []byte{
			0x7b, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		},
		brightness: 0x00,
		padding:    0x00,
		log:        log,
	}
}

func (c *Brightness) GetBytes() [][]byte {
	tmp := utils.BZero(250, c.padding)
	cmd := append(c.bytes, c.brightness)
	copy(tmp, cmd)
	return [][]byte{tmp}
}

func (c *Brightness) SetCount(count int64) {
	_ = count
}

func (c *Brightness) GetName() string {
	return c.name
}

func (c *Brightness) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  0,
		Bytes: nil,
	}
}

func (c *Brightness) ValidateCommand([]byte, int) error {
	return nil
}

func (c *Brightness) SetBrightness(value int) *Brightness {
	v := byte((float64(value) / 100.0) * 255)
	c.brightness = v
	return c
}
