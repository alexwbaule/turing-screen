package theme


type Network struct {
	Wifi     *NetworkMesurement `mapstructure:"WLO"`
	Wired    *NetworkMesurement `mapstructure:"ETH"`
}

type NetworkMesurement struct {
	Upload     *Upload     `mapstructure:"UPLOAD"`
	Download   *Download   `mapstructure:"DOWNLOAD"`
	Uploaded   *Uploaded   `mapstructure:"UPLOADED"`
	Downloaded *Downloaded `mapstructure:"DOWNLOADED"`
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
