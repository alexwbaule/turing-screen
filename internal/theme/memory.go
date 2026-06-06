package theme

type Memory struct {
	Swap     *MemMeasurement `yaml:"SWAP,omitempty"`
	Virtual  *MemMeasurement `yaml:"VIRTUAL,omitempty"`
}

type MemMeasurement struct {
	Show        bool    `yaml:"SHOW"`
	Graph       *Graph  `yaml:"GRAPH,omitempty"`
	Radial      *Radial `yaml:"RADIAL,omitempty"`
	Chart       *Chart  `yaml:"CHART,omitempty"`
	Used        *Text   `yaml:"USED,omitempty"`
	Free        *Text   `yaml:"FREE,omitempty"`
	PercentText *Text   `yaml:"PERCENT_TEXT,omitempty"`
}
