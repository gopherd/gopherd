package backend

import (
	"github.com/gopherd/doge/service/component"

	"github.com/gopherd/gopherd/cmd/gated/module"
	"github.com/gopherd/gopherd/cmd/gated/module/backend/internal"
)

type Component interface {
	component.Component
	module.Backend
}

type Service = internal.Service

// NewComponent creates Backend component
func NewComponent(service internal.Service) Component {
	return internal.NewComponent(service)
}
