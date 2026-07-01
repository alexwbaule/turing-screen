package theme

type Network struct {
	Wifi  *NetworkMesurement `mapstructure:"WLO"`
	Wired *NetworkMesurement `mapstructure:"ETH"`
}

type NetworkMesurement struct {
	Upload     *Sensor `mapstructure:"UPLOAD"`
	Download   *Sensor `mapstructure:"DOWNLOAD"`
	Uploaded   *Sensor `mapstructure:"UPLOADED"`
	Downloaded *Sensor `mapstructure:"DOWNLOADED"`
}
