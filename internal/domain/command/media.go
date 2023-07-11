package command

import (
	"bytes"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
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
	log     *logger.Logger
}

func NewMedia(log *logger.Logger) *Media {
	return &Media{
		log: log,
	}
}

func (m *Media) GetBytes() [][]byte {
	tmp := utils.BZero(250, m.padding)
	copy(tmp, m.bytes)
	return [][]byte{tmp}
}

func (m *Media) GetName() string {
	return m.name
}

func (m *Media) GetSize() int {
	return m.size
}

func (m *Media) ValidateCommand(s []byte, i int) error {
	v := string(bytes.Trim(s, "\x00"))
	if i == m.size && m.readed.MatchString(v) {
		return nil
	}
	return fmt.Errorf("no matching item on: %s", m.readed.String())
}

func (m *Media) StopVideo() *Media {
	return &Media{
		name: "STOP_VIDEO",
		bytes: []byte{
			0x79, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		log:     m.log,
	}
}
func (m *Media) StopMedia() *Media {
	return &Media{
		name: "STOP_MEDIA",
		bytes: []byte{
			0x96, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		size:    1024,
		readed:  mediaStop,
		log:     m.log,
	}
}
func (m *Media) QueryStatus() *Media {
	return &Media{
		name: "QUERY_STATUS",
		bytes: []byte{
			0xcf, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		size:    1024,
		readed:  render,
		log:     m.log,
	}
}
func (m *Media) PostUpdateBitmap() *Media {
	return &Media{
		name: "PRE_UPDATE_BITMAP",
		bytes: []byte{
			0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		log:     m.log,
	}
}
