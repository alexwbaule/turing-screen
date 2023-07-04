package device

import (
	"bytes"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
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
	log     *logger.Logger
}

func NewDevice(log *logger.Logger) *Device {
	return &Device{
		log: log,
	}
}

func (d *Device) GetBytes() [][]byte {
	tmp := utils.BZero(250, d.padding)
	copy(tmp, d.bytes)
	return [][]byte{tmp}
}

func (d *Device) GetName() string {
	return d.name
}

func (d *Device) GetSize() int {
	return d.size
}

func (d *Device) ValidateCommand(s []byte, i int) error {
	v := string(bytes.Trim(s, "\x00"))
	if i == d.size && d.readed.MatchString(v) {
		return nil
	}
	return fmt.Errorf("no matching item on: %s", d.readed.String())
}

func (d *Device) Hello() *Device {
	return &Device{
		name: "HELLO",
		bytes: []byte{
			0x01, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0xc5, 0xd3,
		},
		padding: 0x00,
		size:    23,
		readed:  deviceId,
		log:     d.log,
	}
}

func (d *Device) Restart() *Device {
	return &Device{
		name: "RESTART",
		bytes: []byte{
			0x84, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		log:     d.log,
	}
}
func (d *Device) TurnOff() *Device {
	return &Device{
		name: "TURNOFF",
		bytes: []byte{
			0x83, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		log:     d.log,
	}
}

func (d *Device) RestartScreen() *Device {
	return &Device{
		name: "RESTARTSCREEN",
		bytes: []byte{
			0x84, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		log:     d.log,
	}
}
