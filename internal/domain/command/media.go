package command

import (
	"bytes"
	"errors"
	"regexp"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
)

var (
	mediaStop   = regexp.MustCompile(`^media_stop$`)
	queryStatus = regexp.MustCompile(`^standby$`) // A generic response, can be adjusted
	ErrMatch    = errors.New("no matching item")
)

type Media struct {
	bytes   []byte
	payload []byte
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
	finalCmd := make([]byte, len(m.bytes)+len(m.payload))
	copy(finalCmd, m.bytes)
	if m.payload != nil {
		copy(finalCmd[len(m.bytes):], m.payload)
	}

	tmp := utils.BZero(250, m.padding)
	copy(tmp, finalCmd)
	return [][]byte{tmp}
}

func (m *Media) SetCount(count int64) {
	_ = count
}

func (m *Media) GetName() string {
	return m.name
}

func (m *Media) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  m.size,
		Bytes: nil,
	}
}

func (m *Media) ValidateCommand(s []byte, i int) error {
	if m.readed == nil {
		return nil // No validation needed
	}
	v := string(bytes.Trim(s, "\x00"))
	if i >= 0 && m.readed.MatchString(v) { // Changed to i >= 0 to handle empty but valid responses
		return nil
	}
	return ErrMatch
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

func (m *Media) StartVideo(videoPath string) *Media {
	path := utils.ParsePath(videoPath)

	pyd := []byte{byte(len(path))}
	pyd = append(pyd, PADDING_NULL[0], PADDING_NULL[0], PADDING_NULL[0])
	pyd = append(pyd, []byte(path)...)

	return &Media{
		name: "START_VIDEO",
		bytes: append([]byte{
			0x78, 0xef, 0x69, 0x00, 0x00, 0x00}, pyd...,
		),
		padding: 0x00,
		log:     m.log,
	}
}

func (m *Media) StartDisplayBitmap() *Media {
	return &Media{
		name:    "START_DISPLAY_BITMAP",
		bytes:   []byte{0x2c},
		padding: 0x2c,
		log:     m.log,
	}
}

func (m *Media) DisplayBitmapOnVideo() *Media {
	return &Media{
		name: "DISPLAY_BITMAP_ON_VIDEO",
		bytes: []byte{
			0xca, 0xef, 0x69, 0x00, 0x17, 0x70,
		},
		padding: 0x00,
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
		readed:  nil, // No specific validation, just read
		log:     m.log,
	}
}
