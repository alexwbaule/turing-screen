package command

import (
	"bytes"
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"

	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"regexp"
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

	for i := 0; i < size; i += chunk {
		end := i + chunk
		if end > size {
			end = size
		}
		tmp := utils.BZero(250, m.padding[1])
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

func (m *Payload) SendPayload(background device.ImageBackground) *Payload {
	return &Payload{
		name: "SEND_PAYLOAD",
		bytes: [][]byte{
			{
				0x2c,
			},
			{
				0xc8, 0xef, 0x69, 0x00, 0x17, 0x70,
			},
		},
		padding: []byte{0x2c, 0x00},
		payload: background.GenerateBackgroundImage(m.orientation),
		size:    1024,
		readed:  imageSucess,
		log:     m.log,
	}
}
