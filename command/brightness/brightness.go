package brightness

import (
	"encoding/hex"
	"fmt"
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

func (c *Brightness) GetBytes() []byte {
	tmp := utils.BZero(250, c.padding)
	cmd := append(c.bytes, c.brightness)
	copy(tmp, cmd)
	fmt.Printf("%d - [%s]\n", len(tmp), hex.EncodeToString(tmp))
	return tmp
}

func (c *Brightness) GetName() string {
	return c.name
}

func (c *Brightness) SetBrightness(value int) Brightness {
	return Brightness{
		name: "SetBrightness",
		bytes: []byte{
			0x7b, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		},
		brightness: byte((value / 100) * 255),
		padding:    0x00,
	}
}
