package theme

type Memory struct {
	Model   *Sensor `mapstructure:"MODEL"`
	Total   *Sensor `mapstructure:"TOTAL"`
	Swap    *Sensor `mapstructure:"SWAP"`
	Virtual *Sensor `mapstructure:"VIRTUAL"`
}
