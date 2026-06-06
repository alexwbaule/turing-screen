package theme

type GPU struct {
	Percentage  *Measurement `yaml:"PERCENTAGE,omitempty"`
	Memory      *Measurement `yaml:"MEMORY,omitempty"`
	Temperature *Measurement `yaml:"TEMPERATURE,omitempty"`
	Power       *Measurement `yaml:"POWER,omitempty"`
	Frequency   *Measurement `yaml:"FREQUENCY,omitempty"`
	Voltage     *Measurement `yaml:"VOLTAGE,omitempty"`
	Fan         *Measurement `yaml:"FAN,omitempty"`
}
