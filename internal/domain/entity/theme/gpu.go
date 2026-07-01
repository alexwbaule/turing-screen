package theme

type GPU struct {
	Model       *Sensor `mapstructure:"MODEL"`
	Percentage  *Sensor `mapstructure:"PERCENTAGE"`
	Memory      *Sensor `mapstructure:"MEMORY"`
	Temperature *Sensor `mapstructure:"TEMPERATURE"`
	Power       *Sensor `mapstructure:"POWER"`
	Frequency   *Sensor `mapstructure:"FREQUENCY"`
	Voltage     *Sensor `mapstructure:"VOLTAGE"`
	Fan         *Sensor `mapstructure:"FAN"`
}
