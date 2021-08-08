package oos

import (
	"github.com/gopherd/doge/service/module"
	"github.com/gopherd/gopherd/auth"
)

func New(service auth.Service) interface {
	module.Module
	auth.OOSModule
} {
	return newOOSModule(service)
}

// oosModule implements auth.OOSModule
type oosModule struct {
	module.BaseModule
	service auth.Service
}

func newOOSModule(service auth.Service) *oosModule {
	return &oosModule{
		service: service,
	}
}

func (mod *oosModule) GetObject(obj interface{}, where ...auth.Cond) (bool, error) {
	panic("TODO")
}

func (mod *oosModule) HasObject(key string, where ...auth.Cond) (bool, error) {
	panic("TODO")
}

func (mod *oosModule) InsertObject(obj interface{}) error {
	panic("TODO")
}

func (mod *oosModule) UpdateObject(obj interface{}, fields ...string) (int64, error) {
	panic("TODO")
}
