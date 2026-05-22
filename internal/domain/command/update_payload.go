package command

import (
	"bytes"
	"errors"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
	tdevice "github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/alexwbaule/turing-screen/internal/domain/entity/theme"

	"math/big"
	"regexp"

	"github.com/alexwbaule/turing-screen/internal/resource/process/device"
)

// ErrPayloadTooLarge is returned when the payload size exceeds the maximum
// supported by the 3-byte size field (16,777,215 bytes).
var ErrPayloadTooLarge = errors.New("payload size exceeds maximum of 16,777,215 bytes for UPDATE_BITMAP")

var (
	render = regexp.MustCompile(`^needReSend:0\|renderCnt:0$`)
)

type UpdatePayload struct {
	bytes        []byte
	payload      []byte
	name         string
	padding      byte
	size         int
	count        int64
	readed       *regexp.Regexp
	log          *logger.Logger
	orientation  theme.Orientation
	device       *tdevice.Display
	regionX      int
	regionY      int
	regionWidth  int
	regionHeight int
}

func NewUpdatePayload(log *logger.Logger, o theme.Orientation, t *tdevice.Display) *UpdatePayload {
	log.Infof("NewUpdatePayload: %d", 0)
	return &UpdatePayload{
		log:         log,
		orientation: o,
		device:      t,
	}
}

func (m *UpdatePayload) GetBytes() [][]byte {
	var fullImage [][]byte
	var updateBitMapCmd = utils.BZero(250, m.padding)

	size := len(m.payload)
	pSize := utils.PadBegin(big.NewInt(int64(size)).Bytes(), 3)
	pCount := utils.PadBegin(big.NewInt(m.count).Bytes(), 4)
	pPad := make([]byte, 3)

	copy(updateBitMapCmd, m.bytes)
	copy(updateBitMapCmd[4:], pSize)
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
	m.count = count
}

func (m *UpdatePayload) GetName() string {
	return m.name
}

func (m *UpdatePayload) ValidateWrite() WriteValidation {
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
	v := string(bytes.Trim(s, "\x00"))
	if i == m.size && m.readed.MatchString(v) {
		return nil
	}
	return ErrMatch
}

// GetRegion returns the region key for this UPDATE_BITMAP command,
// enabling per-region deduplication in the RegionQueue.
func (m *UpdatePayload) GetRegion() RegionKey {
	return RegionKey{
		X:      m.regionX,
		Y:      m.regionY,
		Width:  m.regionWidth,
		Height: m.regionHeight,
	}
}

func (m *UpdatePayload) SendPayload(partial device.ImagePartial, x, y int, encoding PixelEncoding) (*UpdatePayload, error) {
	payload := partial.GeneratePartialImage(m.orientation, m.device, x, y, device.PixelEncoding(encoding))

	if len(payload) > 16_777_215 {
		return nil, ErrPayloadTooLarge
	}

	// Extract image dimensions from the partial image for region identification
	width, height := partial.GetDimensions()

	return &UpdatePayload{
		name: "UPDATE_BITMAP",
		bytes: []byte{
			0xcc, 0xef, 0x69, 0x00,
		},
		padding:      0x00,
		size:         1024,
		readed:       render,
		payload:      payload,
		log:          m.log,
		regionX:      x,
		regionY:      y,
		regionWidth:  width,
		regionHeight: height,
	}, nil
}
