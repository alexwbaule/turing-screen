package command

// PixelEncoding defines the pixel format used for partial display updates.
type PixelEncoding int

const (
	EncodingBGR  PixelEncoding = iota // 3 bytes per pixel (Blue, Green, Red), ROM <= 88
	EncodingBGRA                      // 4 bytes per pixel (Blue, Green, Red, Alpha), ROM > 88
)

// EncodingConfig holds the pixel encoding mode determined by the device ROM version.
type EncodingConfig struct {
	Mode       PixelEncoding
	ROMVersion int
}

// NewEncodingConfig creates an EncodingConfig based on the device ROM version.
// ROM versions greater than 88 use BGRA encoding (4 bytes per pixel).
// ROM versions 88 or below use BGR encoding (3 bytes per pixel).
func NewEncodingConfig(romVersion int) *EncodingConfig {
	mode := EncodingBGR
	if romVersion > 88 {
		mode = EncodingBGRA
	}

	return &EncodingConfig{
		Mode:       mode,
		ROMVersion: romVersion,
	}
}
