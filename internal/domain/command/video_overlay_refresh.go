package command

import (
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
)

// VideoOverlayRefresh represents an UPDATE_BITMAP command with diff payload.
// The payload is already pre-formatted by the overlay diff engine with
// 4-bit pixel packing, position encoding, and boundary fix.
//
// GetBytes() returns (matching reference ResfreshVideoOverlay):
//   - Packet 1: header (18 bytes) padded to 250
//   - Packet 2: entire payload padded to next 250 boundary (ONE block)
//
// No response is expected (ValidateWrite.Size = 0).
type VideoOverlayRefresh struct {
	name    string
	header  []byte
	payload []byte
	padding byte
	log     *logger.Logger
}

func NewVideoOverlayRefresh(log *logger.Logger, header []byte, payload []byte) *VideoOverlayRefresh {
	return &VideoOverlayRefresh{
		name:    "VIDEO_OVERLAY_REFRESH",
		header:  header,
		payload: payload,
		padding: 0x00,
		log:     log,
	}
}

func (c *VideoOverlayRefresh) GetBytes() [][]byte {
	// Packet 1: header padded to 250
	hdrPkt := utils.BZero(250, c.padding)
	copy(hdrPkt, c.header)

	// Packet 2: entire payload padded to next 250 boundary (ONE block).
	// The reference sendCommand(SEND_PAYLOAD, imgPayload, ...) pads the
	// payload to 250 boundary at the END, not split into 250-byte chunks.
	payloadSize := len(c.payload)
	paddedSize := payloadSize
	if paddedSize%250 != 0 {
		paddedSize = ((payloadSize / 250) + 1) * 250
	}
	payloadPkt := utils.BZero(paddedSize, c.padding)
	copy(payloadPkt, c.payload)

	return [][]byte{hdrPkt, payloadPkt}
}

func (c *VideoOverlayRefresh) SetCount(count int64) {
	_ = count
}

func (c *VideoOverlayRefresh) GetName() string {
	return c.name
}

func (c *VideoOverlayRefresh) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  0,
		Bytes: nil,
	}
}

func (c *VideoOverlayRefresh) ValidateCommand([]byte, int) error {
	return nil
}
