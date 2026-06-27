package config

import (
	"fmt"

	"github.com/alexwbaule/turing-screen/internal/domain/entity/device"
	"github.com/spf13/viper"
)

type Config struct {
	device *device.Config
}

const defaultConfig = `conf/config.yaml`

func NewDefaultConfig() (*Config, error) {
	var config device.Config

	viper.SetConfigType("yaml")
	viper.SetConfigFile(defaultConfig)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config file: %w", err)
	}
	return &Config{
		device: &config,
	}, err
}

func (c *Config) GetLogLevel() string {
	return c.device.LogLevel
}
func (c *Config) GetDevicePort() string {
	return c.device.Port
}
func (c *Config) GetThemeName() string {
	return c.device.Theme
}
func (c *Config) GetDeviceDisplay() *device.Display {
	return &c.device.Display
}
func (c *Config) GetNetworkConfig() device.Net {
	return c.device.Sensors.Net
}
func (c *Config) GetCPUSensorConfig() device.CPUSensorConfig {
	return c.device.Sensors.CPU
}
func (c *Config) GetDiskSensorConfig() device.DiskSensorConfig {
	return c.device.Sensors.Disk
}
func (c *Config) GetGPUSensorConfig() device.GPUSensorConfig {
	return c.device.Sensors.GPU
}
func (c *Config) GetWeatherConfig() *device.WeatherConfig {
	return &c.device.Sensors.Weather
}
func (c *Config) GetMemoryConfig() device.MemoryConfig {
	return c.device.Sensors.Memory
}
func (c *Config) GetTurnOffOnExit() bool {
	return c.device.TurnOffOnExit
}
func (c *Config) GetAPIPort() int {
	if c.device.APIPort <= 0 {
		return 9120
	}
	return c.device.APIPort
}

func (c *Config) GetDebugConfig() device.Debug {
	return c.device.Debug
}

// Reload re-reads the config file from disk.
func (c *Config) Reload() error {
	var config device.Config
	viper.SetConfigType("yaml")
	viper.SetConfigFile(defaultConfig)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("error unmarshalling config file: %w", err)
	}
	c.device = &config
	return nil
}

// WriteThemeName persists device.theme to the config file on disk.
//
// The daemon reloads the theme from conf/config.yaml whenever it restarts the
// sensor loop (see controller.ApplyTheme), so any caller that wants a theme
// switch to survive the restart must write the name here first. This keeps the
// GUI as a pure socket client: it only sends theme.apply and the daemon owns
// its own config persistence.
//
// Uses a standalone viper instance so it never disturbs the global viper state
// that the getters above rely on.
func WriteThemeName(name string) error {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(defaultConfig)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}
	v.Set("device.theme", name)
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}
	return nil
}

