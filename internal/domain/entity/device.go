package entity

type Config struct {
	Device
}
type Device struct {
	Port  string `mapstructure:"port"`
	Theme string `mapstructure:"theme"`
	Sensors
	Display
}

type Net struct {
	Wired string `mapstructure:"eth"`
	Wifi  string `mapstructure:"wlo"`
}

type Sensors struct {
	Net `mapstructure:"network"`
}

type Display struct {
	Reverse    bool `mapstructure:"reverse"`
	Brightness int  `mapstructure:"brightness"`
}
