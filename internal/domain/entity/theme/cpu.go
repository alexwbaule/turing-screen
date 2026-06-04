package theme

import "time"

type CPU struct {
	Interval    time.Duration `mapstructure:"INTERVAL"`
	Percentage  *Mesurement   `mapstructure:"PERCENTAGE"`
	Frequency   *Mesurement   `mapstructure:"FREQUENCY"`
	Load        *Load         `mapstructure:"LOAD"`
	Temperature *Mesurement   `mapstructure:"TEMPERATURE"`
	Fan         *Mesurement   `mapstructure:"FAN"`
	Power       *Mesurement   `mapstructure:"POWER"`
	Voltage     *Mesurement   `mapstructure:"VOLTAGE"`
}

type LoadOne struct {
	Text *Text `mapstructure:"TEXT"`
}
type LoadFive struct {
	Text *Text `mapstructure:"TEXT"`
}
type LoadFifteen struct {
	Text *Text `mapstructure:"TEXT"`
}
type Load struct {
	Interval time.Duration `mapstructure:"INTERVAL"`
	One      *LoadOne      `mapstructure:"ONE"`
	Five     *LoadFive     `mapstructure:"FIVE"`
	Fifteen  *LoadFifteen  `mapstructure:"FIFTEEN"`
}
