package main

import (
	"github.com/gopherd/doge/service"
	_ "github.com/gopherd/redis/discovery"

	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/server"
)

func main() {
	service.Run(server.New(config.New()))
}
