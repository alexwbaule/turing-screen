package command

import (
	"bytes"
	"image"
	"math/big"
	"regexp"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	tdevice "github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"
	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
	"github.com/alexwbaule/turing-screen/internal/utils"
)

var (
	render = regexp.MustCompile(`^needReSend:0\|renderCnt:\d+$`)
)

// OverlayWriter abstracts the video overlay buffer operations.
type OverlayWriter interface {
	Draw(img image.Image, x, y int)
	Refresh() Command
}

// UpdatePayload handles sensor update commands.
// In static mode: generates UPDATE_BITMAP commands sent directly to device.
// In video mode: draws on the overlay buffer (actual send is via refresh goroutine).
type UpdatePayload struct {
	bytes       []byte
	payload     []byte
	name        string
	source      string
	padding     byte
	size        int
	count       int64
	readed      *regexp.Regexp
	log         *logger.Logger
	orientation theme.Orientation
	device      *tdevice.Display
	overlay     OverlayWriter
	inner       Command
}

func NewUpdatePayload(log *logger.Logger, o theme.Orientation, t *tdevice.Display) *UpdatePayload {
	return &UpdatePayload{
		log:         log,
		orientation: o,
		device:      t,
	}
}

// SetOverlay activates video mode.
func (m *UpdatePayload) SetOverlay(overlay OverlayWriter) {
	m.overlay = overlay
}

// IsVideoMode returns true if the overlay is active (video mode).
func (m *UpdatePayload) IsVideoMode() bool {
	return m.overlay != nil
}

// SendPayload creates an UPDATE_BITMAP command for static mode,
// or draws on the overlay buffer for video mode (returning NoOp).
func (m *UpdatePayload) SendPayload(partial device.ImagePartial, x, y int) *UpdatePayload {
	return m.SendPayloadFrom(partial, x, y, "")
}

// SendPayloadFrom creates an UPDATE_BITMAP with a source label for debug logging.
func (m *UpdatePayload) SendPayloadFrom(partial device.ImagePartial, x, y int, source string) *UpdatePayload {
	if m.overlay != nil {
		return m.drawOnOverlay(partial, x, y)
	}
	up := m.sendStaticUpdate(partial, x, y)
	up.source = source
	return up
}

// drawOnOverlay draws the partial image on the video overlay buffer.
// Returns a NoOp command — the actual send happens via the refresh goroutine.
func (m *UpdatePayload) drawOnOverlay(partial device.ImagePartial, x, y int) *UpdatePayload {
	transformedImg, tx, ty := partial.TransformForOverlay(m.orientation, m.device, x, y)
	m.overlay.Draw(transformedImg, tx, ty)
	return &UpdatePayload{
		inner: NewNoOp(),
		log:   m.log,
	}
}

// sendStaticUpdate generates an UPDATE_BITMAP command for static mode.
func (m *UpdatePayload) sendStaticUpdate(partial device.ImagePartial, x, y int) *UpdatePayload {
	return &UpdatePayload{
		name: "UPDATE_BITMAP",
		bytes: []byte{
			0xcc, 0xef, 0x69, 0x00, 0x00,
		},
		padding: 0x00,
		size:    1024,
		readed:  render,
		payload: partial.GeneratePartialImage(m.orientation, m.device, x, y),
		log:     m.log,
	}
}

func (m *UpdatePayload) GetBytes() [][]byte {
	if m.inner != nil {
		return m.inner.GetBytes()
	}

	var fullImage [][]byte
	var updateBitMapCmd = utils.BZero(250, m.padding)

	size := len(m.payload)
	pSize := utils.PadBegin(big.NewInt(int64(size)).Bytes(), 2)
	pCount := utils.PadBegin(big.NewInt(m.count).Bytes(), 4)
	pPad := make([]byte, 3)

	copy(updateBitMapCmd, m.bytes)
	copy(updateBitMapCmd[5:], pSize)
	copy(updateBitMapCmd[7:], pPad)
	copy(updateBitMapCmd[10:], pCount)

	fullImage = append(fullImage, updateBitMapCmd)
	for i := 0; i < size; i += chunk {
		end := i + chunk
		if end > size {
			end = size
		}
		tmp := utils.BZero(250, m.padding)
		copy(tmp, m.payload[i:end])
		fullImage = append(fullImage, tmp)
	}
	return fullImage
}

func (m *UpdatePayload) SetCount(count int64) {
	if m.inner != nil {
		m.inner.SetCount(count)
		return
	}
	m.count = count
}

func (m *UpdatePayload) GetName() string {
	if m.inner != nil {
		return m.inner.GetName()
	}
	if m.source != "" {
		return m.name + " [" + m.source + "]"
	}
	return m.name
}

func (m *UpdatePayload) ValidateWrite() WriteValidation {
	if m.inner != nil {
		return m.inner.ValidateWrite()
	}
	return WriteValidation{
		Size:  m.size,
		Bytes: m.QueryStatus(),
	}
}

func (m *UpdatePayload) QueryStatus() []byte {
	tmp := utils.BZero(250, 0x00)
	copy(tmp, []byte{0xcf, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01})
	return tmp
}

func (m *UpdatePayload) ValidateCommand(s []byte, i int) error {
	if m.inner != nil {
		return m.inner.ValidateCommand(s, i)
	}
	v := string(bytes.Trim(s[:i], "\x00"))
	if m.readed.MatchString(v) {
		return nil
	}
	return ErrMatch
}
