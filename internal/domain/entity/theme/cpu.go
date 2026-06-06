package theme


type CPU struct {
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
	One      *LoadOne      `mapstructure:"ONE"`
	Five     *LoadFive     `mapstructure:"FIVE"`
	Fifteen  *LoadFifteen  `mapstructure:"FIFTEEN"`
}
