package theme

type DateTime struct {
	Day  *Sensor `mapstructure:"DAY"`
	Hour *Sensor `mapstructure:"HOUR"`
}
