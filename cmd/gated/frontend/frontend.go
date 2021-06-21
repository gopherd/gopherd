package frontend

import (
	"github.com/gopherd/doge/service/component"

	"github.com/gopherd/gopherd/cmd/gated/frontend/internal"
	"github.com/gopherd/gopherd/cmd/gated/module"
)

// Component declares frontend component interface
type Component interface {
	component.Component
	module.Frontend
}

// Service aliases service for frontend component
type Service = internal.Service

// NewComponent creates Frontent component
func NewComponent(service internal.Service) Component {
	return internal.NewComponent(service)
}
