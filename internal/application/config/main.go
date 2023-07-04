package config

import (
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	*viper.Viper
}

const defaultConfig = `conf/config.yaml`

func NewConfig(file string) (*Config, error) {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	v.AutomaticEnv()
	v.SetConfigType("yaml")
	v.SetConfigFile(file)
	f := v.ReadInConfig()
	return &Config{
		v,
	}, f
}

func NewDefaultConfig() (*Config, error) {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	v.AutomaticEnv()
	v.SetConfigType("yaml")
	v.SetConfigFile(defaultConfig)
	f := v.ReadInConfig()
	return &Config{
		v,
	}, f
}
