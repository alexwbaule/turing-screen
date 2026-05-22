package command

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"

	"github.com/alexwbaule/turing-screen/internal/application/logger"
	"github.com/alexwbaule/turing-screen/internal/application/utils"
)

var (
	deviceId       = regexp.MustCompile(`^chs_5inch.dev1_rom\d.\d{2}`)
	romVersionExpr = regexp.MustCompile(`chs_5inch\.dev1_rom(\d)\.(\d{2})`)
)

// HelloResponse holds the parsed result of a HELLO handshake response.
type HelloResponse struct {
	ROMVersion int
	RawString  string
}

// ParseHelloResponse extracts the ROM version integer from the HELLO response.
// The expected pattern is `chs_5inch.dev1_romX.YY` where YY is the version number.
// If parsing fails, returns a default HelloResponse with ROMVersion = 99 (triggers BGRA)
// and logs a warning.
func ParseHelloResponse(response []byte, log *logger.Logger) (*HelloResponse, error) {
	raw := string(bytes.Trim(response, "\x00"))

	matches := romVersionExpr.FindStringSubmatch(raw)
	if len(matches) < 3 {
		log.Warnf("could not parse ROM version from HELLO response: %q, defaulting to ROM version 99 (BGRA)", raw)
		return &HelloResponse{
			ROMVersion: 99,
			RawString:  raw,
		}, nil
	}

	version, err := strconv.Atoi(matches[2])
	if err != nil {
		log.Warnf("could not convert ROM version to integer: %q, defaulting to ROM version 99 (BGRA)", matches[2])
		return &HelloResponse{
			ROMVersion: 99,
			RawString:  raw,
		}, nil
	}

	return &HelloResponse{
		ROMVersion: version,
		RawString:  raw,
	}, nil
}

type Device struct {
	bytes   []byte
	name    string
	padding byte
	size    int
	readed  *regexp.Regexp
	log     *logger.Logger
}

func NewDevice(log *logger.Logger) *Device {
	return &Device{
		log: log,
	}
}

func (d *Device) GetBytes() [][]byte {
	tmp := utils.BZero(250, d.padding)
	copy(tmp, d.bytes)
	return [][]byte{tmp}
}

func (d *Device) SetCount(count int64) {
	_ = count
}

func (d *Device) GetName() string {
	return d.name
}

func (d *Device) ValidateWrite() WriteValidation {
	return WriteValidation{
		Size:  d.size,
		Bytes: nil,
	}
}

func (d *Device) ValidateCommand(s []byte, i int) error {
	v := string(bytes.Trim(s, "\x00"))
	if i == d.size && d.readed.MatchString(v) {
		return nil
	}
	return fmt.Errorf("no matching item on: %s", d.readed.String())
}

func (d *Device) Hello() *Device {
	return &Device{
		name: "HELLO",
		bytes: []byte{
			0x01, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0xc5, 0xd3,
		},
		padding: 0x00,
		size:    23,
		readed:  deviceId,
		log:     d.log,
	}
}

func (d *Device) Restart() *Device {
	return &Device{
		name: "RESTART",
		bytes: []byte{
			0x84, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		log:     d.log,
	}
}
func (d *Device) TurnOff() *Device {
	return &Device{
		name: "TURNOFF",
		bytes: []byte{
			0x83, 0xef, 0x69, 0x00, 0x00, 0x00, 0x01,
		},
		padding: 0x00,
		log:     d.log,
	}
}
