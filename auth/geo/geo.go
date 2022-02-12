package geo

import (
	"errors"
	"net"
	"strings"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/service/module"
	"github.com/oschwald/geoip2-golang"

	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/config"
)

var errInvalidIP = errors.New("invalid ip")

type Service interface {
	Config() *config.Config
}

func New(service Service) interface {
	module.Module
	auth.GeoModule
} {
	return newGeoModule(service)
}

// geoModule implements auth.GeoModule
type geoModule struct {
	*module.BasicModule
	service Service
	db      *geoip2.Reader
}

func newGeoModule(service Service) *geoModule {
	return &geoModule{
		BasicModule: module.NewBasicModule("geo"),
		service:    service,
	}
}

func (mod *geoModule) Init() error {
	if err := mod.BasicModule.Init(); err != nil {
		return err
	}
	filepath := mod.service.Config().GeoIP.Filepath
	db, err := geoip2.Open(filepath)
	if err != nil {
		return erron.Throwf("load geoip from %q error: %w", filepath, err)
	}
	mod.db = db
	return nil
}

func (mod *geoModule) Shutdown() {
	defer mod.BasicModule.Shutdown()
	if mod.db != nil {
		mod.db.Close()
	}
}

func (mod *geoModule) QueryLocation(ip, lang string) (country, province, city string, err error) {
	if lang == "" || strings.HasPrefix(lang, "en-") {
		lang = "en"
	}
	x := net.ParseIP(ip)
	if len(x) == 0 || x.IsUnspecified() {
		err = errInvalidIP
		return
	}
	var r *geoip2.City
	r, err = mod.db.City(x)
	if err != nil {
		return
	}
	city = r.City.Names[lang]
	if len(r.Subdivisions) > 0 {
		province = r.Subdivisions[0].Names[lang]
	}
	country = r.Country.Names[lang]
	return
}
