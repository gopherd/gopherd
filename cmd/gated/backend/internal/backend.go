package internal

import (
	"github.com/gopherd/jwt"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/service/component"

	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/module"
)

type Service interface {
	GetConfig() *config.Config
	Frontend() module.Frontend
}

func NewComponent(service Service) *backend {
	return newBackend(service)
}

type backend struct {
	*component.BaseComponent
	service Service
}

func newBackend(service Service) *backend {
	return &backend{
		BaseComponent: component.NewBaseComponent("backend"),
		service:       service,
	}
}

func (b *backend) Forward(uid int64, typ proto.Type, body proto.Body) error {
	return nil
}

func (b *backend) Login(uid int64, claims *jwt.Claims, userdata []byte) error {
	return nil
}

func (b *backend) Logout(uid int64) error {
	return nil
}
