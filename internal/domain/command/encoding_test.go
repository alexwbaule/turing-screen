package command

import "testing"

func TestNewEncodingConfig_BGRForROM88(t *testing.T) {
	cfg := NewEncodingConfig(88)
	if cfg.Mode != EncodingBGR {
		t.Errorf("expected EncodingBGR for ROM 88, got %d", cfg.Mode)
	}
	if cfg.ROMVersion != 88 {
		t.Errorf("expected ROMVersion 88, got %d", cfg.ROMVersion)
	}
}

func TestNewEncodingConfig_BGRForROMLessThan88(t *testing.T) {
	cfg := NewEncodingConfig(50)
	if cfg.Mode != EncodingBGR {
		t.Errorf("expected EncodingBGR for ROM 50, got %d", cfg.Mode)
	}
	if cfg.ROMVersion != 50 {
		t.Errorf("expected ROMVersion 50, got %d", cfg.ROMVersion)
	}
}

func TestNewEncodingConfig_BGRAForROMGreaterThan88(t *testing.T) {
	cfg := NewEncodingConfig(89)
	if cfg.Mode != EncodingBGRA {
		t.Errorf("expected EncodingBGRA for ROM 89, got %d", cfg.Mode)
	}
	if cfg.ROMVersion != 89 {
		t.Errorf("expected ROMVersion 89, got %d", cfg.ROMVersion)
	}
}

func TestNewEncodingConfig_BGRAForROM99(t *testing.T) {
	cfg := NewEncodingConfig(99)
	if cfg.Mode != EncodingBGRA {
		t.Errorf("expected EncodingBGRA for ROM 99, got %d", cfg.Mode)
	}
}

func TestNewEncodingConfig_BGRForROM0(t *testing.T) {
	cfg := NewEncodingConfig(0)
	if cfg.Mode != EncodingBGR {
		t.Errorf("expected EncodingBGR for ROM 0, got %d", cfg.Mode)
	}
}
