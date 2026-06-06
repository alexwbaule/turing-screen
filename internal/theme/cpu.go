package theme

type CPU struct {
	Percentage  *Measurement `yaml:"PERCENTAGE,omitempty"`
	Frequency   *Measurement `yaml:"FREQUENCY,omitempty"`
	Load        *Load        `yaml:"LOAD,omitempty"`
	Temperature *Measurement `yaml:"TEMPERATURE,omitempty"`
	Fan         *Measurement `yaml:"FAN,omitempty"`
	Power       *Measurement `yaml:"POWER,omitempty"`
	Voltage     *Measurement `yaml:"VOLTAGE,omitempty"`
}

type LoadOne struct {
	Text *Text `yaml:"TEXT"`
}
type LoadFive struct {
	Text *Text `yaml:"TEXT"`
}
type LoadFifteen struct {
	Text *Text `yaml:"TEXT"`
}
type Load struct {
	One      *LoadOne     `yaml:"ONE"`
	Five     *LoadFive    `yaml:"FIVE"`
	Fifteen  *LoadFifteen `yaml:"FIFTEEN"`
}
