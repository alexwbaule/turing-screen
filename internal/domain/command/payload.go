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
	imageSucess   = regexp.MustCompile("^full_png_sucess$")
	overlaySucess = regexp.MustCompile("^seq_png_init_sucess$")
	renderSucess  = regexp.MustCompile("^needReSend:0\\|renderCnt:0$")
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

	// Se payload vazio, não itera e retorna o header já feito
	if size == 0 {
		return fullImage
	}

	for i := 0; i < size; i += chunk {
		end := i + chunk
		if end > size {
			end = size
		}
		// A padding rule depends on the second element for chunking
		padVal := byte(0x00)
		if len(m.padding) > 1 {
			padVal = m.padding[1]
		}
		tmp := utils.BZero(250, padVal)
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
	// Some commands like INIT_VIDEO_OVERLAY don't need a QUERY_STATUS directly inside Payload if handled by queue/worker
	if m.size == 0 {
		return WriteValidation{
			Size:  0,
			Bytes: nil,
		}
	}
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
	if i == m.size && m.readed != nil && m.readed.MatchString(v) {
		return nil
	}
	if m.readed == nil {
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

// SendOverlay sends the background image as a video overlay using the 0xca command.
// This is used after PLAY_VIDEO to render sensor data on top of the playing video.
// The device responds with "seq_png_init_sucess" instead of "full_png_sucess".
func (m *Payload) SendOverlay(background device.ImageBackground) *Payload {
	return &Payload{
		name: "SEND_OVERLAY",
		bytes: [][]byte{
			{
				0x2c,
			},
			{
				0xca, 0xef, 0x69, 0x00, 0x17, 0x70,
			},
		},
		padding: []byte{0x2c, 0x00},
		payload: background.GenerateBackgroundImage(m.orientation),
		size:    1024,
		readed:  overlaySucess,
		log:     m.log,
	}
}

// InitVideoOverlay sends the end marker and initializes the video overlay on the device.
// This is required to signal the device which pixels are visible. In this case,
// only the end marker is sent for initialization, effectively rendering everything
// non-transparent over the video.
func (m *Payload) InitVideoOverlay() *Payload {
	return &Payload{
		name: "INIT_VIDEO_OVERLAY",
		bytes: [][]byte{
			{
				0xd0, 0xef, 0x69, 0x00, 0x00, 0x00, 0x02,
			},
			{
				0xef, 0x69,
			},
		},
		padding: []byte{0x00, 0x00},
		payload: nil,
		size:    0,
		readed:  nil,
		log:     m.log,
	}
}
