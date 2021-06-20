package backend

import (
	"github.com/gopherd/doge/service/component"

	"github.com/gopherd/gopherd/cmd/gated/module"
	"github.com/gopherd/gopherd/cmd/gated/module/backend/internal"
)

// Component declares backend component interface
type Component interface {
	component.Component
	module.Backend
}

// Service aliases service for backend component
type Service = internal.Service

// NewComponent creates backend component
func NewComponent(service internal.Service) Component {
	return internal.NewComponent(service)
}
