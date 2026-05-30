package command

import (
	"bytes"
	"fmt"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"

	"regexp"

	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
)

var (
	imageSucess = regexp.MustCompile("^full_png_sucess$")
)

type Payload struct {
	bytes       [][]byte
	payload     []byte
	name        string
	padding     []byte
	size        int
	readed      *regexp.Regexp
	log         *logger.Logger
	orientation theme.Orientation
}

func NewPayload(log *logger.Logger, o theme.Orientation) *Payload {
	return &Payload{
		log:         log,
		orientation: o,
	}
}

func (m *Payload) GetBytes() [][]byte {
	var fullImage [][]byte

	for i, b := range m.bytes {
		tmp := utils.BZero(250, m.padding[i])
		copy(tmp, b)
		fullImage = append(fullImage, tmp)
	}
	size := len(m.payload)

	// The padding for the payload chunks should be 0x00, which is the third byte in the padding slice.
	payloadPadding := m.padding[2]

	for i := 0; i < size; i += chunk {
		end := i + chunk
		if end > size {
			end = size
		}
		tmp := utils.BZero(250, payloadPadding)
		copy(tmp, m.payload[i:end])
		fullImage = append(fullImage, tmp)
	}
	return fullImage
}

func (m *Payload) SetCount(count int64) {
	_ = count
}

func (m *Payload) GetName() string {
	return m.name
}

func (m *Payload) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  m.size,
		Bytes: m.QueryStatus(),
	}
}
func (m *Payload) QueryStatus() []byte {
	tmp := utils.BZero(250, 0x00)
	copy(tmp, []byte{0xcf, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01})
	return tmp
}

func (m *Payload) ValidateCommand(s []byte, i int) error {
	v := string(bytes.Trim(s, "\x00"))
	if i == m.size && m.readed.MatchString(v) {
		return nil
	}
	return fmt.Errorf("no matching item on: %s", m.readed.String())
}

func (m *Payload) SendPayload(background device.ImageBackground, isVideoOverlay bool) *Payload {
	var displayCmd []byte
	var cmdName string
	var size int
	if isVideoOverlay {
		cmdName = "SEND_PAYLOAD_VIDEO"
		// DISPLAY_BITMAP_ON_VIDEO
		displayCmd = []byte{0xca, 0xef, 0x69, 0x00, 0x17, 0x70}
		size = 0
	} else {
		cmdName = "SEND_PAYLOAD_STATIC"
		// DISPLAY_BITMAP
		displayCmd = []byte{0xc8, 0xef, 0x69, 0x00, 0x17, 0x70}
		size = 0
	}

	return &Payload{
		name: cmdName,
		bytes: [][]byte{
			{
				0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, // PRE_UPDATE_BITMAP
			},
			{
				0x2c, // START_DISPLAY_BITMAP
			},
			displayCmd,
		},
		padding: []byte{0x00, 0x2c, 0x00}, // Padding for each command: 0x00 for PRE_UPDATE, 0x2c for START_DISPLAY, 0x00 for payload chunks
		payload: background.GenerateBackgroundImage(m.orientation),
		size:    size,
		readed:  imageSucess,
		log:     m.log,
	}
}
