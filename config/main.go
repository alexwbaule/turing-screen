package config

import (
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	*viper.Viper
}

const defaultConfig = `conf/config.yaml`

func NewConfig(file *string) (*Config, error) {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	v.AutomaticEnv()
	v.SetConfigType("yaml")

	if file == nil {
		v.SetConfigFile(defaultConfig)
	} else {
		v.SetConfigFile(*file)
	}

	return &Config{
		v,
	}, v.ReadInConfig()
}
