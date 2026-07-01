package theme

type CPU struct {
	Model       *Sensor `mapstructure:"MODEL"`
	Percentage  *Sensor `mapstructure:"PERCENTAGE"`
	Frequency   *Sensor `mapstructure:"FREQUENCY"`
	Temperature *Sensor `mapstructure:"TEMPERATURE"`
	Fan         *Sensor `mapstructure:"FAN"`
	Power       *Sensor `mapstructure:"POWER"`
	Voltage     *Sensor `mapstructure:"VOLTAGE"`
}
