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
func (c *Config) GetDeviceDisplay() device.Display {
	return c.device.Display
}
func (c *Config) GetNetworkConfig() device.Net {
	return c.device.Sensors.Net
}
