package brightness

import (
	"github.com/alexwbaule/turing-screen/utils"
)

type Brightness struct {
	name       string
	bytes      []byte
	brightness byte
	padding    byte
}

func NewBrightness() *Brightness {
	return &Brightness{}
}

func (c *Brightness) GetBytes() [][]byte {
	tmp := utils.BZero(250, c.padding)
	cmd := append(c.bytes, c.brightness)
	copy(tmp, cmd)
	return [][]byte{tmp}
}

func (c *Brightness) GetName() string {
	return c.name
}

func (c *Brightness) GetSize() int {
	return 0
}

func (c *Brightness) ValidateCommand([]byte, int) error {
	return nil
}

func (c *Brightness) SetBrightness(value int) *Brightness {
	v := byte((float64(value) / 100.0) * 255)
	return &Brightness{
		name: "SetBrightness",
		bytes: []byte{
			0x7b, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		},
		brightness: v,
		padding:    0x00,
	}
}
