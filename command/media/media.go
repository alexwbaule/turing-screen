package media

import (
	"encoding/hex"
	"fmt"
	"github.com/alexwbaule/turing-screen/utils"
	"regexp"
)

var (
	mediaStop = regexp.MustCompile(`^media_stop$`)
	render    = regexp.MustCompile(`^needReSend:0\|renderCnt:0$`)
)

type Media struct {
	bytes   []byte
	name    string
	padding byte
	size    int
	readed  *regexp.Regexp
}

func NewMedia() *Media {
	return &Media{}
}

func (m *Media) GetBytes() []byte {
	tmp := utils.BZero(250, m.padding)
	copy(tmp, m.bytes)
	fmt.Printf("%d - [%s]\n", len(tmp), hex.EncodeToString(tmp))
	return tmp
}

func (m *Media) GetName() string {
	return m.name
}

func (m *Media) GetSize() int {
	return m.size
}

func (m *Media) ValidateCommand(s []byte, i int) error {
	if i == m.size && m.readed.MatchString(string(s)) {
		return nil
	}
	return fmt.Errorf("no matching item on: %s", m.readed.String())
}

func (m *Media) StartDisplayBitmap() Media {
	return Media{
		name: "START_DISPLAY_BITMAP",
		bytes: []byte{
			0x2c,
		},
		padding: 0x2c,
	}
}

func (m *Media) StopVideo() Media {
	return Media{
		name: "STOP_VIDEO",
		bytes: []byte{
			0x79, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
	}
}
func (m *Media) StopMedia() Media {
	return Media{
		name: "STOP_MEDIA",
		bytes: []byte{
			0x96, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		size:    1024,
		readed:  mediaStop,
	}
}
func (m *Media) QueryStatus() Media {
	return Media{
		name: "QUERY_STATUS",
		bytes: []byte{
			0xcf, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		size:    1024,
		readed:  render,
	}
}
func (m *Media) PreUpdateBitmap() Media {
	return Media{
		name: "PRE_UPDATE_BITMAP",
		bytes: []byte{
			0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
	}
}
func (m *Media) UpdateBitmap() Media {
	return Media{
		name: "UPDATE_BITMAP",
		bytes: []byte{
			0xcc, 0xef, 0x69, 0x00, 0x00,
		},
		padding: 0x00,
	}

}
func (m *Media) DisplayBitmap() Media {
	return Media{
		name: "DISPLAY_BITMAP",
		bytes: []byte{
			0xc8, 0xef, 0x69, 0x00, 0x17, 0x70,
		},
		padding: 0x00,
	}
}

func (m *Media) SendPayload() Media {
	return Media{
		name: "SEND_PAYLOAD",
		bytes: []byte{
			0xff,
		},
	}
}
