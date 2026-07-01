package theme

type Host struct {
	Hostname *Sensor `mapstructure:"HOSTNAME"`
	Load     *Load   `mapstructure:"LOAD"`
}

type Load struct {
	One     *Sensor `mapstructure:"ONE"`
	Five    *Sensor `mapstructure:"FIVE"`
	Fifteen *Sensor `mapstructure:"FIFTEEN"`
}
