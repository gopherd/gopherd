package config

import (
	"os"
	"path/filepath"

	"github.com/gopherd/doge/config"
	"github.com/gopherd/doge/net/httputil"
)

type Config struct {
	config.BaseConfig

	AccessTokenTTL  int64             `json:"access_token_ttl"`
	RefreshTokenTTL int64             `json:"refresh_token_ttl"`
	HTTP            httputil.Config   `json:"http"`
	Proviers        map[string]string `json:"providers"`

	JWT struct {
		Filename string `json:"filename"`
		Issuer   string `json:"issuer"`
		KeyId    string `json:"key_id"`
	} `json:"jwt"`

	GeoLite struct {
		Filepath string `json:"filepath"`
	} `json:"geo_lite"`

	Routers struct {
		Authorize string `json:"authorize"` // default: /auth/authorize
		Link      string `json:"link"`      // default: /auth/link
		SMSCode   string `json:"smscode"`   // default: /auth/smscode
	} `json:"routers"`
}

// Default implements config.Configurator Default method
func (*Config) Default() config.Configurator {
	c := &Config{
		Proviers: make(map[string]string),
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	c.GeoLite.Filepath = filepath.Join(home, "geoip", "GeoLite2-City", "GeoLite2-City.mmdb")
	return c
}
