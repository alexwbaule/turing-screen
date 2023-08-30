package theme

import "time"

type Disk struct {
	Interval    time.Duration `mapstructure:"INTERVAL"`
	Used        *Mesurement   `mapstructure:"USED"`
	Total       *Mesurement   `mapstructure:"TOTAL"`
	Free        *Mesurement   `mapstructure:"FREE"`
	Temperature *Mesurement   `mapstructure:"TEMPERATURE"`
}
