package theme

import "time"

type Weather struct {
	Temperature *Mesurement   `mapstructure:"TEMPERATURE"`
	Condition   *Text         `mapstructure:"CONDITION"`
	Enabled     bool          `mapstructure:"ENABLED"`
	City        string        `mapstructure:"CITY"`
	ApiKey      string        `mapstructure:"API_KEY"`
	Interval    time.Duration `mapstructure:"INTERVAL"`
}
