package command

import (
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
)

// VideoOverlayRefresh represents an UPDATE_BITMAP command with diff payload.
// The payload is already pre-formatted by the overlay diff engine with
// 249-byte data chunks separated by 0x00 bytes (matching the reference
// ResfreshVideoOverlay's sendCommand(SEND_PAYLOAD, ...) behavior).
//
// GetBytes() returns:
//   - Packet 1: header (18 bytes) padded to 250
//   - Packets 2..N: payload in 250-byte aligned chunks (NOT 249)
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
	var packets [][]byte

	// Packet 1: header padded to 250
	hdrPkt := utils.BZero(250, c.padding)
	copy(hdrPkt, c.header)
	packets = append(packets, hdrPkt)

	// Packets 2..N: payload in 250-byte aligned chunks.
	// The payload from Refresh() already has 249+1 structure (249 data + 0x00 sep).
	// We must split on 250-byte boundaries to preserve that structure,
	// matching the reference's sendCommand(SEND_PAYLOAD, imgPayload, ...) which
	// pads the entire payload to 250 boundaries at the end.
	size := len(c.payload)
	for i := 0; i < size; i += 250 {
		end := i + 250
		if end > size {
			end = size
		}
		pkt := utils.BZero(250, c.padding)
		copy(pkt, c.payload[i:end])
		packets = append(packets, pkt)
	}

	return packets
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
