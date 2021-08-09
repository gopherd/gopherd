package oos

import (
	"github.com/gopherd/doge/service/module"
	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/config"
)

type Service interface {
	Config() *config.Config
}

func New(service Service) interface {
	module.Module
	auth.OOSModule
} {
	return newOOSModule(service)
}

// oosModule implements auth.OOSModule
type oosModule struct {
	module.BaseModule
	service Service
}

func newOOSModule(service Service) *oosModule {
	return &oosModule{
		service: service,
	}
}

func (mod *oosModule) GetObject(obj interface{}, where ...auth.Field) (bool, error) {
	panic("TODO")
}

func (mod *oosModule) HasObject(key string, where ...auth.Field) (bool, error) {
	panic("TODO")
}

func (mod *oosModule) InsertObject(obj interface{}) error {
	panic("TODO")
}

func (mod *oosModule) UpdateObject(obj interface{}, fields ...string) (int64, error) {
	panic("TODO")
}
