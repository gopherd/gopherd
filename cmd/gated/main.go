package main

import (
	"github.com/gopherd/doge/service"

	// drivers
	_ "github.com/gopherd/redis/discovery"
	_ "github.com/gopherd/redis/mq"
	_ "github.com/gopherd/zmq"

	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/server"
)

func main() {
	service.Run(server.New(config.New()))
}
