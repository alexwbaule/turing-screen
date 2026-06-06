package theme


type GPU struct {
	Percentage  *Mesurement   `mapstructure:"PERCENTAGE"`
	Memory      *Mesurement   `mapstructure:"MEMORY"`
	Temperature *Mesurement   `mapstructure:"TEMPERATURE"`
	Power       *Mesurement   `mapstructure:"POWER"`
	Frequency   *Mesurement   `mapstructure:"FREQUENCY"`
	Voltage     *Mesurement   `mapstructure:"VOLTAGE"`
	Fan         *Mesurement   `mapstructure:"FAN"`
}
