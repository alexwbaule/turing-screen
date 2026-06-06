package theme

type Network struct {
	Wifi     *NetworkMeasurement `yaml:"WLO,omitempty"`
	Wired    *NetworkMeasurement `yaml:"ETH,omitempty"`
}

type NetworkMeasurement struct {
	Upload     *Measurement `yaml:"UPLOAD,omitempty"`
	Download   *Measurement `yaml:"DOWNLOAD,omitempty"`
	Uploaded   *Measurement `yaml:"UPLOADED,omitempty"`
	Downloaded *Measurement `yaml:"DOWNLOADED,omitempty"`
}

type Upload struct {
	Text *Text `yaml:"TEXT"`
}
type Download struct {
	Text *Text `yaml:"TEXT"`
}
type Uploaded struct {
	Text *Text `yaml:"TEXT"`
}
type Downloaded struct {
	Text *Text `yaml:"TEXT"`
}
