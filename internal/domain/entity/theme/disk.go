package theme

type Disk struct {
	Model       *Sensor `mapstructure:"MODEL"`
	Used        *Sensor `mapstructure:"USED"`
	Total       *Sensor `mapstructure:"TOTAL"`
	Free        *Sensor `mapstructure:"FREE"`
	Temperature *Sensor `mapstructure:"TEMPERATURE"`
}
