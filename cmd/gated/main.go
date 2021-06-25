package main

import (
	"github.com/gopherd/doge/service"

	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/server"

	// imports drivers
	_ "github.com/gopherd/redis/discovery"
	_ "github.com/gopherd/redis/redismq"
	_ "github.com/gopherd/zmq"
)

func main() {
	service.Run(server.New(config.New()))
}
