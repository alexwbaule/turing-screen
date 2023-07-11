package theme

import "time"

type Memory struct {
	INTERVAL time.Duration `mapstructure:"INTERVAL"`
	SWAP     *Swap         `mapstructure:"SWAP"`
	VIRTUAL  *Virtual      `mapstructure:"VIRTUAL"`
}

type Swap struct {
	Graph  *Graph  `mapstructure:"GRAPH"`
	Radial *Radial `mapstructure:"RADIAL"`
}
type Virtual struct {
	Graph       *Graph  `mapstructure:"GRAPH"`
	Radial      *Radial `mapstructure:"RADIAL"`
	Used        *Text   `mapstructure:"USED"`
	Free        *Text   `mapstructure:"FREE"`
	PercentText *Text   `mapstructure:"PERCENT_TEXT"`
}
