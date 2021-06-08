package main

import (
	"github.com/gopherd/doge/service"

	"github.com/gopherd/gopherd/cmd/gated/server"
)

func main() {
	service.Run(server.New())
}
