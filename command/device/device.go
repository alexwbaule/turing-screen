package device

import (
	"encoding/hex"
	"fmt"
	"github.com/alexwbaule/turing-screen/utils"
	"regexp"
)

var (
	deviceId = regexp.MustCompile(`^chs_5inch.dev1_rom\d.\d{2}`)
)

type Device struct {
	bytes   []byte
	name    string
	padding byte
	size    int
	readed  *regexp.Regexp
}

func NewDevice() *Device {
	return &Device{}
}

func (d *Device) GetBytes() []byte {
	tmp := utils.BZero(250, d.padding)
	copy(tmp, d.bytes)
	fmt.Printf("%d - [%s]\n", len(tmp), hex.EncodeToString(tmp))
	return tmp
}

func (d *Device) GetName() string {
	return d.name
}

func (d *Device) GetSize() int {
	return d.size
}

func (d *Device) ValidateCommand(s []byte, i int) error {
	if i == d.size && d.readed.MatchString(string(s)) {
		return nil
	}
	return fmt.Errorf("no matching item on: %s", d.readed.String())
}

func (d *Device) Hello() Device {
	return Device{
		name: "HELLO",
		bytes: []byte{
			0x01, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0xc5, 0xd3,
		},
		padding: 0x00,
		size:    23,
		readed:  deviceId,
	}
}

func (d *Device) Restart() Device {
	return Device{
		name: "RESTART",
		bytes: []byte{
			0x84, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
	}
}
func (d *Device) TurnOff() Device {
	return Device{
		name: "TURNOFF",
		bytes: []byte{
			0x83, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
	}
}

func (d *Device) RestartScreen() Device {
	return Device{
		name: "RESTARTSCREEN",
		bytes: []byte{
			0x84, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
	}
}
