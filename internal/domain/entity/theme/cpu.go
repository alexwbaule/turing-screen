package theme

type CPU struct {
	Model       *Sensor `mapstructure:"MODEL"`
	Percentage  *Sensor `mapstructure:"PERCENTAGE"`
	Frequency   *Sensor `mapstructure:"FREQUENCY"`
	Load        *Load   `mapstructure:"LOAD"`
	Temperature *Sensor `mapstructure:"TEMPERATURE"`
	Fan         *Sensor `mapstructure:"FAN"`
	Power       *Sensor `mapstructure:"POWER"`
	Voltage     *Sensor `mapstructure:"VOLTAGE"`
}

type Load struct {
	One     *Sensor `mapstructure:"ONE"`
	Five    *Sensor `mapstructure:"FIVE"`
	Fifteen *Sensor `mapstructure:"FIFTEEN"`
}
