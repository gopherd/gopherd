package frontend

import (
	"github.com/gopherd/doge/service/component"

	"github.com/gopherd/gopherd/cmd/gated/module"
	"github.com/gopherd/gopherd/cmd/gated/module/frontend/internal"
)

// Component is the interface that groups ther basic Component and FooComponent methods
type Component interface {
	component.Component
	module.Frontend
}

// Component is the interface that groups ther basic Component and BarComponent methods
func NewComponent(service internal.Service) Component {
	return internal.NewComponent(service)
}
