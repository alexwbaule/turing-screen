package command

import (
	"encoding/hex"
	"fmt"
	"github.com/alexwbaule/turing-screen/utils"
	"regexp"
)

var (
	deviceId  = regexp.MustCompile(`^chs_5inch.dev1_rom\d.\d{2}`)
	mediaStop = regexp.MustCompile(`^media_stop$`)
	render    = regexp.MustCompile(`^needReSend:0\|renderCnt:0$`)
)

type Command struct {
	bytes   []byte
	name    string
	padding byte
	size    int
	readed  *regexp.Regexp
}

func NewCommand() *Command {
	return &Command{}
}

func (c Command) GetBytes() []byte {
	tmp := utils.BZero(250, c.padding)
	copy(tmp, c.bytes)
	fmt.Printf("%d - [%s]\n", len(tmp), hex.EncodeToString(tmp))
	return tmp
}

func (c Command) GetName() string {
	return c.name
}

func (c Command) ValidateCommand(s []byte, i int) bool {
	if i == c.size && c.readed.MatchString(string(s)) {
		return true
	}
	return false
}

func (c Command) StartDisplayBitmap() Command {
	return Command{
		name:    "START_DISPLAY_BITMAP",
		bytes:   []byte{0x2c},
		padding: 0x2c,
	}
}

func (c Command) Hello() Command {
	return Command{
		name: "HELLO",
		bytes: []byte{
			0x01, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0xc5, 0xd3,
		},
		padding: 0x00,
		size:    23,
		readed:  deviceId,
	}
}

func (c Command) Restart() Command {
	return Command{
		name: "RESTART",
		bytes: []byte{
			0x84, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
	}
}
func (c Command) TurnOff() Command {
	return Command{
		name: "TURNOFF",
		bytes: []byte{
			0x83, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
	}
}

func (c Command) StopVideo() Command {
	return Command{
		name: "STOP_VIDEO",
		bytes: []byte{
			0x79, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
	}
}
func (c Command) StopMedia() Command {
	return Command{
		name: "STOP_MEDIA",
		bytes: []byte{
			0x96, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		size:    1024,
		readed:  mediaStop,
	}
}
func (c Command) QueryStatus() Command {
	return Command{
		name: "QUERY_STATUS",
		bytes: []byte{
			0xcf, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		size:    1024,
		readed:  render,
	}
}
func (c Command) PreUpdateBitmap() Command {
	return Command{
		name: "PRE_UPDATE_BITMAP",
		bytes: []byte{
			0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
	}
}
func (c Command) UpdateBitmap() Command {
	return Command{
		name: "UPDATE_BITMAP",
		bytes: []byte{
			0xcc, 0xef, 0x69, 0x00, 0x00,
		},
		padding: 0x00,
	}

}
func (c Command) RestartScreen() Command {
	return Command{
		name: "RESTARTSCREEN",
		bytes: []byte{
			0x84, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
	}
}
func (c Command) DisplayBitmap() Command {
	return Command{
		name: "DISPLAY_BITMAP",
		bytes: []byte{
			0xc8, 0xef, 0x69, 0x00, 0x17, 0x70,
		},
		padding: 0x00,
	}
}

func (c Command) SendPayload() Command {
	return Command{
		name: "SEND_PAYLOAD",
		bytes: []byte{
			0xff,
		},
	}
}
