package config

import (
	"fmt"
	"github.com/alexwbaule/turing-screen/internal/domain/entity"
	"github.com/spf13/viper"
)

type Config struct {
	device *entity.Config
}

const defaultConfig = `conf/config.yaml`

func NewDefaultConfig() (*Config, error) {
	var config entity.Config

	viper.SetConfigType("yaml")
	viper.SetConfigFile(defaultConfig)
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Error: [%#v]\n", err)
		return nil, err
	}
	err = viper.Unmarshal(&config)
	if err != nil {
		fmt.Printf("Error: [%#v]\n", err)
		return nil, err
	}
	return &Config{
		device: &config,
	}, err
}

func (c *Config) GetDevicePort() string {
	return c.device.Port
}

func (c *Config) GetDeviceTheme() string {
	return c.device.Theme
}

func (c *Config) GetDeviceDisplay() entity.Display {
	return c.device.Display
}
