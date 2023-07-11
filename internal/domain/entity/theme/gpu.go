package theme

import "time"

type GPU struct {
	INTERVAL    time.Duration   `mapstructure:"INTERVAL"`
	PERCENTAGE  *Mesurement     `mapstructure:"PERCENTAGE"`
	MEMORY      *Mesurement     `mapstructure:"MEMORY"`
	TEMPERATURE *GpuTemperature `mapstructure:"TEMPERATURE"`
}

type GpuTemperature struct {
	Text *Text `mapstructure:"TEXT"`
}
