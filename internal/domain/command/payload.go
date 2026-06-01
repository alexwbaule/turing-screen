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
	v := string(bytes.Trim(s[:i], "\x00"))
	if m.readed != nil && m.readed.MatchString(v) {
		return nil
	}
	if m.readed == nil {
		return nil
	}
	return fmt.Errorf("no matching item on: %s, got: %q", m.readed.String(), v)
}

// SendStaticBitmap sends the full background image for static mode.
// Protocol: PRE_UPDATE_BITMAP (0x86) → SEPARATOR (0x2c) → DISPLAY_BITMAP (0xc8) → payload
// Response: "full_png_sucess"
func (m *Payload) SendStaticBitmap(background device.ImageBackground) *Payload {
	return &Payload{
		name: "SEND_STATIC_BITMAP",
		bytes: [][]byte{
			{0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01}, // PRE_UPDATE_BITMAP
			{0x2c},                               // START_DISPLAY_BITMAP
			{0xc8, 0xef, 0x69, 0x00, 0x17, 0x70}, // DISPLAY_BITMAP
		},
		padding: []byte{0x00, 0x2c, 0x00},
		payload: background.GenerateBackgroundImage(m.orientation),
		size:    1024,
		readed:  imageSucess,
		log:     m.log,
	}
}

// SendVideoOverlay sends the overlay image for video mode.
// Protocol: PRE_UPDATE_BITMAP (0x86) → SEPARATOR (0x2c) → DISPLAY_BITMAP_ON_VIDEO (0xca) → payload
// No response expected (size=0) — the InitVideoOverlay handles the response.
func (m *Payload) SendVideoOverlay(background device.ImageBackground) *Payload {
	return &Payload{
		name: "SEND_VIDEO_OVERLAY",
		bytes: [][]byte{
			{0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01}, // PRE_UPDATE_BITMAP
			{0x2c},                               // START_DISPLAY_BITMAP
			{0xca, 0xef, 0x69, 0x00, 0x17, 0x70}, // DISPLAY_BITMAP_ON_VIDEO
		},
		padding: []byte{0x00, 0x2c, 0x00},
		payload: background.GenerateBackgroundImage(m.orientation),
		size:    0,
		readed:  nil,
		log:     m.log,
	}
}

// SendPayload is kept for backward compatibility. Use SendStaticBitmap or SendVideoOverlay instead.
func (m *Payload) SendPayload(background device.ImageBackground, isVideoOverlay bool) *Payload {
	if isVideoOverlay {
		return m.SendVideoOverlay(background)
	}
	return m.SendStaticBitmap(background)
}
