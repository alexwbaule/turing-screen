package theme

type Disk struct {
	Used        *Measurement `yaml:"USED,omitempty"`
	Total       *Measurement `yaml:"TOTAL,omitempty"`
	Free        *Measurement `yaml:"FREE,omitempty"`
	Temperature *Measurement `yaml:"TEMPERATURE,omitempty"`
}
