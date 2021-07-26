package config

import (
	"github.com/gopherd/doge/config"
)

// Config represents config of gated service
type Config struct {
	config.BaseConfig

	Net struct {
		// Protocol: websocket/tcp
		Protocol    string `json:"protocol"`
		Bind        string `json:"bind"`
		Port        int    `json:"port"`
		Keepalive   int    `json:"keepalive"`    // seconds
		ReadTimeout int    `json:"read_timeout"` // seconds
	} `json:"net"`
	Keepalive                   int    `json:"keepalive"`
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
	return new(Config)
}
