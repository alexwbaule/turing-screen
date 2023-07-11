package theme

import "time"

type CPU struct {
	Interval    time.Duration `mapstructure:"INTERVAL"`
	Percentage  *Mesurement   `mapstructure:"PERCENTAGE"`
	Frequency   *Mesurement   `mapstructure:"FREQUENCY"`
	Load        *Load         `mapstructure:"LOAD"`
	Temperature *Mesurement   `mapstructure:"TEMPERATURE"`
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
	Interval *int         `mapstructure:"INTERVAL"`
	One      *LoadOne     `mapstructure:"ONE"`
	Five     *LoadFive    `mapstructure:"FIVE"`
	Fifteen  *LoadFifteen `mapstructure:"FIFTEEN"`
}
