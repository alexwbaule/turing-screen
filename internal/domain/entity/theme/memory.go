package theme

// RAM holds both the usage widgets (graph, radial, percent_text …) for physical
// RAM and the two static-string sub-sensors: MODEL (e.g. "DDR4-3200") and SIZE
// (e.g. "16 GB", sourced from hwinfo at runtime).
type RAM struct {
	Graph       *Graph     `mapstructure:"GRAPH"`
	Radial      *Radial    `mapstructure:"RADIAL"`
	Gauge       *Gauge     `mapstructure:"GAUGE"`
	StatusBar   *StatusBar `mapstructure:"STATUS_BAR"`
	Chart       *Chart     `mapstructure:"CHART"`
	Text        *Text      `mapstructure:"TEXT"`
	PercentText *Text      `mapstructure:"PERCENT_TEXT"`
	Model       *Sensor    `mapstructure:"MODEL"`
	Size        *Sensor    `mapstructure:"SIZE"`
}

type Memory struct {
	RAM  *RAM    `mapstructure:"RAM"`
	Swap *Sensor `mapstructure:"SWAP"`
}
