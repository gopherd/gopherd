package frontend

import (
	"github.com/gopherd/doge/service/component"

	"github.com/gopherd/gopherd/cmd/gated/module"
	"github.com/gopherd/gopherd/cmd/gated/module/frontend/internal"
)

type Component interface {
	component.Component
	module.Frontend
}

type Service = internal.Service

// NewComponent creates Frontent component
func NewComponent(service internal.Service) Component {
	return internal.NewComponent(service)
}
