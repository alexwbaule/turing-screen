package theme

import "time"

type Disk struct {
	Interval time.Duration `mapstructure:"INTERVAL"`
	Used     *Mesurement   `mapstructure:"USED"`
	Total    *Total        `mapstructure:"TOTAL"`
	Free     *Free         `mapstructure:"FREE"`
}

type Total struct {
	Text *Text `mapstructure:"TEXT"`
}
type Free struct {
	Text *Text `mapstructure:"TEXT"`
}
