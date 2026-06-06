package theme


type DateTime struct {
	Day      *Day          `mapstructure:"DAY"`
	Hour     *Hour         `mapstructure:"HOUR"`
}

type Day struct {
	Text *Text `mapstructure:"TEXT"`
}
type Hour struct {
	Text *Text `mapstructure:"TEXT"`
}
