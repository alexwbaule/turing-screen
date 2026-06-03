package theme

import "time"

type Weather struct {
	Temperature *Mesurement   `mapstructure:"TEMPERATURE"`
	Condition   *Text         `mapstructure:"CONDITION"`
	Enabled     bool          `mapstructure:"ENABLED"`
	City        string        `mapstructure:"CITY"`
	Interval    time.Duration `mapstructure:"INTERVAL"`
}
