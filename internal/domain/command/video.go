package command

import (
	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/utils"
)

// InitVideoOverlay represents the full video overlay initialization sequence.
// It assembles the protocol packets from pre-encoded BGRA payload bytes.
//
// Packet sequence (matching reference InitializeVideoOverlay):
//
//	PRE_UPDATE_BITMAP → START_DISPLAY_BITMAP → DISPLAY_BITMAP_ON_VIDEO →
//	BGRA payload chunks (250-byte) → INIT_VIDEO_OVERLAY → visible pixels
type InitVideoOverlay struct {
	name    string
	log     *logger.Logger
	packets [][]byte
}

// NewInitVideoOverlay creates the init video overlay command from pre-encoded
// BGRA payload bytes (already rotated and chunked with 249+1 separators).
func NewInitVideoOverlay(log *logger.Logger, bgraPayload []byte) *InitVideoOverlay {
	var packets [][]byte

	// 1. PRE_UPDATE_BITMAP [0x86, 0xEF, 0x69, 0x00, 0x00, 0x00, 0x01] padded to 250
	pkt := utils.BZero(250, 0x00)
	copy(pkt, []byte{0x86, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01})
	packets = append(packets, pkt)

	// 2. START_DISPLAY_BITMAP [0x2c] padded to 250 with 0x2c padding
	pkt = utils.BZero(250, 0x2c)
	copy(pkt, []byte{0x2c})
	packets = append(packets, pkt)

	// 3. DISPLAY_BITMAP_ON_VIDEO [0xCA, 0xEF, 0x69, 0x00, 0x17, 0x70] padded to 250
	pkt = utils.BZero(250, 0x00)
	copy(pkt, []byte{0xca, 0xef, 0x69, 0x00, 0x17, 0x70})
	packets = append(packets, pkt)

	// 4. Split pre-encoded BGRA payload into 250-byte packets
	for i := 0; i < len(bgraPayload); i += 250 {
		end := i + 250
		if end > len(bgraPayload) {
			end = len(bgraPayload)
		}
		pkt = utils.BZero(250, 0x00)
		copy(pkt, bgraPayload[i:end])
		packets = append(packets, pkt)
	}

	// 5. INIT_VIDEO_OVERLAY [0xD0, 0xEF, 0x69, 0x00, 0x00, 0x00, 0x02] padded to 250
	pkt = utils.BZero(250, 0x00)
	copy(pkt, []byte{0xd0, 0xef, 0x69, 0x00, 0x00, 0x00, 0x02})
	packets = append(packets, pkt)

	// 6. SEND_PAYLOAD with visible pixels [0xEF, 0x69] padded to 250
	pkt = utils.BZero(250, 0x00)
	copy(pkt, []byte{0xef, 0x69})
	packets = append(packets, pkt)

	log.Debugf("InitVideoOverlay: %d packets (3 cmds + %d payload + 2 init), bgra=%d bytes",
		len(packets), (len(bgraPayload)+249)/250, len(bgraPayload))

	return &InitVideoOverlay{
		name:    "INIT_VIDEO_OVERLAY",
		log:     log,
		packets: packets,
	}
}

func (c *InitVideoOverlay) GetBytes() [][]byte {
	return c.packets
}

func (c *InitVideoOverlay) SetCount(count int64) {
	_ = count
}

func (c *InitVideoOverlay) GetName() string {
	return c.name
}

func (c *InitVideoOverlay) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  0,
		Bytes: nil,
	}
}

func (c *InitVideoOverlay) ValidateCommand([]byte, int) error {
	return nil
}
