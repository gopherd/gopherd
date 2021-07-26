package config

import (
	"github.com/gopherd/doge/config"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/gopherd/auth"
)

type Config struct {
	config.BaseConfig
	auth.Options

	HTTP     httputil.Config   `json:"http"`
	Proviers map[string]string `json:"providers"`
}

// Default implements config.Configurator Default method
func (*Config) Default() config.Configurator {
	return &Config{
		Proviers: make(map[string]string),
	}
}
