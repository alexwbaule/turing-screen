package command

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
)

var (
	healthCheckResponse = regexp.MustCompile(`^needReSend:0`)
)

// HealthCheck is a command that sends QUERY_STATUS (0xCF) to the device
// as a health probe. It validates that the device responds within the
// serial port's read timeout.
type HealthCheck struct {
	bytes   []byte
	name    string
	padding byte
	size    int
	readed  *regexp.Regexp
	log     *logger.Logger
}

func NewHealthCheck(log *logger.Logger) *HealthCheck {
	return &HealthCheck{
		log: log,
	}
}

// QueryStatus returns a HealthCheck command configured to send the
// QUERY_STATUS probe (0xCF) and validate the device response.
func (h *HealthCheck) QueryStatus() *HealthCheck {
	return &HealthCheck{
		name: "HEALTH_CHECK",
		bytes: []byte{
			0xcf, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		size:    1024,
		readed:  healthCheckResponse,
		log:     h.log,
	}
}

func (h *HealthCheck) GetBytes() [][]byte {
	tmp := utils.BZero(250, h.padding)
	copy(tmp, h.bytes)
	return [][]byte{tmp}
}

func (h *HealthCheck) SetCount(count int64) {
	_ = count
}

func (h *HealthCheck) GetName() string {
	return h.name
}

func (h *HealthCheck) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  h.size,
		Bytes: nil,
	}
}

func (h *HealthCheck) ValidateCommand(s []byte, i int) error {
	v := string(bytes.Trim(s, "\x00"))
	if i == h.size && h.readed.MatchString(v) {
		return nil
	}
	return fmt.Errorf("health check failed: unexpected response %q", v)
}
