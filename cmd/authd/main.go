package main

import (
	"github.com/gopherd/doge/service"

	// drivers
	_ "github.com/gopherd/redis/discovery"
	_ "github.com/gopherd/redis/mq"
	_ "github.com/gopherd/zmq"

	"github.com/gopherd/gopherd/auth/config"
	"github.com/gopherd/gopherd/auth/server"
)

func main() {
	service.Run(server.New(new(config.Config)))
}
