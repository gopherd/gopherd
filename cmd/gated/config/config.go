package config

import (
	"strings"
	"time"

	"github.com/gopherd/doge/config"
)

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
		Key    string `json:"key"`
		Issuer string `json:"issuer"`
	} `json:"jwt"`
	Limiter struct {
		MsgInterval       int `json:"msg_interval"`
		MsgCount          int `json:"msg_count"`
		BroadcastInterval int `json:"broadcast_interval"`
	} `json:"limiter"`
}

func New() *Config {
	return &Config{}
}

const defaultConfigContent = `{
	core: {
		// Unique server id
		id: 1001,

		log: {
			// Log level, supported values: trace,debug,info,warn,error,fatal
			level: "debug",
			// Log prefix
			prefix: "{{.name}}",
			// Log writers
			writers: [
				"consle",
				"file://var/log/{{.name}}.{{.id}}?suffix=.txt"
			]
		},

		mq: {
			name: "zeromq",
			source: "0.0.0.0:2{{.id}}"
		},

		// Service discovery config
		discovery: {
			name: "redis",
			source: "127.0.0.1:2639?key=service.discovery"
		}
	},

	// Listening address
	host: "0.0.0.0",
	port: 1{{.id}}
}
`

func init() {
	cfg := New()
	if err := cfg.Read(cfg, strings.NewReader(defaultConfigContent)); err != nil {
		panic(err)
	}
}
