package theme

type Disk struct {
	Used        *Sensor `mapstructure:"USED"`
	Total       *Sensor `mapstructure:"TOTAL"`
	Free        *Sensor `mapstructure:"FREE"`
	Temperature *Sensor `mapstructure:"TEMPERATURE"`
}
