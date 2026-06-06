package theme

type DateTime struct {
	Day      *Measurement `yaml:"DAY,omitempty"`
	Hour     *Measurement `yaml:"HOUR,omitempty"`
}

type Day struct {
	Text *Text `yaml:"TEXT"`
}
type Hour struct {
	Text *Text `yaml:"TEXT"`
}
