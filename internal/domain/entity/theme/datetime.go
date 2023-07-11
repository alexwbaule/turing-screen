package theme

import "time"

type DateTime struct {
	Interval time.Duration `mapstructure:"INTERVAL"`
	Day      *Day          `mapstructure:"DAY"`
	Hour     *Hour         `mapstructure:"HOUR"`
}

type Day struct {
	Text *Text `mapstructure:"TEXT"`
}
type Hour struct {
	Text *Text `mapstructure:"TEXT"`
}
