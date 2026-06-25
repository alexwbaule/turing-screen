package device

import "time"

// Default intervals for sensors when not configured
const (
	DefaultCPUInterval     = 1 * time.Second
	DefaultGPUInterval     = 1 * time.Second
	DefaultMemoryInterval  = 5 * time.Second
	DefaultDiskInterval    = 10 * time.Second
	DefaultNetworkInterval = 1 * time.Second
	DefaultWeatherInterval = 30 * time.Minute
)

type Config struct {
	Device
}
type Device struct {
	Port          string `mapstructure:"port"`
	Theme         string `mapstructure:"theme"`
	LogLevel      string `mapstructure:"log"`
	TurnOffOnExit bool   `mapstructure:"turn_off_on_exit"`
	APIPort       int    `mapstructure:"api_port"`
	Debug         Debug  `mapstructure:"debug"`
	Sensors
	Display
}

type Debug struct {
	Profiling ProfilingConfig `mapstructure:"profiling"`
}

type ProfilingConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	PprofAddr     string `mapstructure:"pprof_addr"`
	PyroscopeAddr string `mapstructure:"pyroscope_addr"`
}

type Net struct {
	Wired    string        `mapstructure:"eth"`
	Wifi     string        `mapstructure:"wlo"`
	Interval time.Duration `mapstructure:"interval"`
}

func (n Net) GetInterval() time.Duration {
	if n.Interval <= 0 {
		return DefaultNetworkInterval
	}
	return n.Interval
}

type CPUSensorConfig struct {
	TemperatureSensor string        `mapstructure:"temperature_sensor"`
	Interval          time.Duration `mapstructure:"interval"`
}

func (c CPUSensorConfig) GetInterval() time.Duration {
	if c.Interval <= 0 {
		return DefaultCPUInterval
	}
	return c.Interval
}

type DiskSensorConfig struct {
	TemperatureSensor string        `mapstructure:"temperature_sensor"`
	Interval          time.Duration `mapstructure:"interval"`
}

func (d DiskSensorConfig) GetInterval() time.Duration {
	if d.Interval <= 0 {
		return DefaultDiskInterval
	}
	return d.Interval
}

type GPUSensorConfig struct {
	Provider string        `mapstructure:"provider"`
	Interval time.Duration `mapstructure:"interval"`
}

func (g GPUSensorConfig) GetInterval() time.Duration {
	if g.Interval <= 0 {
		return DefaultGPUInterval
	}
	return g.Interval
}

type MemoryConfig struct {
	Interval time.Duration `mapstructure:"interval"`
}

func (m MemoryConfig) GetInterval() time.Duration {
	if m.Interval <= 0 {
		return DefaultMemoryInterval
	}
	return m.Interval
}

type Sensors struct {
	Net     `mapstructure:"network"`
	CPU     CPUSensorConfig  `mapstructure:"cpu"`
	Disk    DiskSensorConfig `mapstructure:"disk"`
	GPU     GPUSensorConfig  `mapstructure:"gpu"`
	Memory  MemoryConfig     `mapstructure:"memory"`
	Weather WeatherConfig    `mapstructure:"weather"`
}

type Display struct {
	Reverse    bool `mapstructure:"reverse"`
	Brightness int  `mapstructure:"brightness"`
	Width      int  `mapstructure:"width"`
	Height     int  `mapstructure:"height"`
}

// WeatherConfig para a configuração do tempo
type WeatherConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	City     string        `mapstructure:"city"`
	Interval time.Duration `mapstructure:"interval"`
}

func (w WeatherConfig) GetInterval() time.Duration {
	if w.Interval <= 0 {
		return DefaultWeatherInterval
	}
	return w.Interval
}
