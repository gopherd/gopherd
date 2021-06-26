package config

import (
	"bytes"
	_ "embed"
	"time"

	"github.com/gopherd/doge/config"
)

//go:embed gated.conf
var defaultConfigContent []byte

// Config represents config of gated service
type Config struct {
	config.BaseConfig

	Net struct {
		Host        string        `json:"host"`
		Port        int           `json:"port"`
		Keepalive   time.Duration `json:"keepalive"`
		ReadTimeout time.Duration `json:"read_timeout"`
	} `json:"net"`
	Keepalive                   int64  `json:"keepalive"`
	ForwardPing                 bool   `json:"forward_ping"`
	UserTTL                     int    `json:"user_ttl"`
	MaxConns                    int    `json:"max_conns"`
	MaxConnsPerIP               int    `json:"max_conns_per_ip"`
	TimeoutForUnauthorizedConn  int    `json:"timeout_for_unauthorized_conn"`
	DefaultLocationForUnknownIP string `json:"default_location_for_unknown_ip"`
	JWT                         struct {
		Filename string `json:"filename"`
		Issuer   string `json:"issuer"`
		KeyId    string `json:"key_id"`
	} `json:"jwt"`
	Limiter struct {
		MsgInterval       int `json:"msg_interval"`
		MsgCount          int `json:"msg_count"`
		BroadcastInterval int `json:"broadcast_interval"`
	} `json:"limiter"`
}

// Default implements config.Configurator Default method
func (*Config) Default() config.Configurator {
	return New()
}

func New() *Config {
	cfg := new(Config)
	if err := cfg.Read(cfg, bytes.NewReader(defaultConfigContent)); err != nil {
		panic(err)
	}
	return cfg
}

func init() {
	// verify default config
	New()
}
