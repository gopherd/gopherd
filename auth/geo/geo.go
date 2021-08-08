package geo

import (
	"github.com/gopherd/doge/service/module"
	"github.com/gopherd/gopherd/auth"
)

func New(service auth.Service) interface {
	module.Module
	auth.GeoModule
} {
	return newGeoModule(service)
}

// geoModule implements auth.GeoModule
type geoModule struct {
	module.BaseModule
	service auth.Service
}

func newGeoModule(service auth.Service) *geoModule {
	return &geoModule{
		service: service,
	}
}

func (mod *geoModule) QueryLocationByIP(ip string) string {
	panic("TODO")
}
