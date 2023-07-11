package theme

import "time"

type Network struct {
	Interval time.Duration      `mapstructure:"INTERVAL"`
	Wifi     *NetworkMesurement `mapstructure:"WLO"`
	Wired    *NetworkMesurement `mapstructure:"ETH"`
}

type NetworkMesurement struct {
	UPLOAD     *Upload     `mapstructure:"UPLOAD"`
	DOWNLOAD   *Download   `mapstructure:"DOWNLOAD"`
	UPLOADED   *Uploaded   `mapstructure:"UPLOADED"`
	DOWNLOADED *Downloaded `mapstructure:"DOWNLOADED"`
}

type Upload struct {
	Text *Text `mapstructure:"TEXT"`
}
type Download struct {
	Text *Text `mapstructure:"TEXT"`
}
type Uploaded struct {
	Text *Text `mapstructure:"TEXT"`
}
type Downloaded struct {
	Text *Text `mapstructure:"TEXT"`
}
