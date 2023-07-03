package payload

import (
	"bytes"
	"fmt"
	"github.com/alexwbaule/turing-screen/image_process"
	"github.com/alexwbaule/turing-screen/utils"
	"regexp"
)

var (
	imageSucess = regexp.MustCompile("^full_png_sucess$")
)

const chunk = 249

type Payload struct {
	bytes   [][]byte
	payload []byte
	name    string
	padding []byte
	size    int
	readed  *regexp.Regexp
}

func NewPayload() *Payload {
	return &Payload{}
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

func (m *Payload) GetName() string {
	return m.name
}

func (m *Payload) GetSize() int {
	return m.size
}

func (m *Payload) ValidateCommand(s []byte, i int) error {
	v := string(bytes.Trim(s, "\x00"))
	if i == m.size && m.readed.MatchString(v) {
		return nil
	}
	return fmt.Errorf("no matching item on: %s", m.readed.String())
}

func (m *Payload) SendPayload(background image_process.ImageBackground) *Payload {
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
		payload: background.GenerateBackgroundImage(),
		size:    1024,
		readed:  imageSucess,
	}
}
