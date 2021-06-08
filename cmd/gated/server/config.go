package server

import (
	"strings"

	"github.com/gopherd/doge/config"
)

// Config represents config of gated service
type Config struct {
	config.BaseConfig

	Host                        string `json:"host"`
	Port                        int    `json:"port"`
	KeepaliveSeconds            int64  `json:"keepalive_seconds"`
	ForwardEcho                 bool   `json:"forward_echo"`
	UserTTL                     int    `json:"user_ttl"`
	MaxConns                    int    `json:"max_conns"`
	MaxConnsPerIP               int    `json:"max_conns_per_ip"`
	TimeoutForUnauthorizedConn  int    `json:"timeout_for_unauthorized_conn"`
	DefaultLocationForUnknownIP string `json:"default_location_for_unknown_ip"`
	JWT                         struct {
		KeyFile string `json:"key_file"`
		Issuer  string `json:"issuer"`
	} `json:"jwt"`
	LimiterMsgInterval       int `json:"limiter_msg_interval"`
	LimiterMsgCount          int `json:"limiter_msg_count"`
	LimiterBroadcastInterval int `json:"limiter_broadcast_interval"`
}

func NewConfig() *Config {
	return &Config{}
}

const defaultConfigContent = `{
	core: {
		id: 1001,

		log: {
			// Log level, supported values: trace,debug,info,warn,error,fatal
			level: "debug",
			// Log prefix
			prefix: "{{.name}}",
			writers: [
				"consle",
				"file://var/log/{{.id}}?suffix=.txt"
			]
		},

		mq: {
			name: "zeromq",
			source: "0.0.0.0:9001"
		},

		discovery: {
			name: "redis",
			source: ""
		}
	}
}
`

func init() {
	cfg := NewConfig()
	if err := cfg.Read(cfg, strings.NewReader(defaultConfigContent)); err != nil {
		panic(err)
	}
}
