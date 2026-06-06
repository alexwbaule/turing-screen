package theme


type Disk struct {
	Used        *Mesurement   `mapstructure:"USED"`
	Total       *Mesurement   `mapstructure:"TOTAL"`
	Free        *Mesurement   `mapstructure:"FREE"`
	Temperature *Mesurement   `mapstructure:"TEMPERATURE"`
}
