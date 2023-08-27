package theme

import "time"

type GPU struct {
	Interval    time.Duration `mapstructure:"INTERVAL"`
	Percentage  *Mesurement   `mapstructure:"PERCENTAGE"`
	Memory      *Mesurement   `mapstructure:"MEMORY"`
	Temperature *Mesurement   `mapstructure:"TEMPERATURE"`
}
