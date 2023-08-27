package theme

import "time"

type Memory struct {
	Interval time.Duration  `mapstructure:"INTERVAL"`
	Swap     *MemMesurement `mapstructure:"SWAP"`
	Virtual  *MemMesurement `mapstructure:"VIRTUAL"`
}

type MemMesurement struct {
	Graph       *Graph  `mapstructure:"GRAPH"`
	Radial      *Radial `mapstructure:"RADIAL"`
	Used        *Text   `mapstructure:"USED"`
	Free        *Text   `mapstructure:"FREE"`
	PercentText *Text   `mapstructure:"PERCENT_TEXT"`
}
