package geo_test

import (
	"testing"

	"github.com/gopherd/gopherd/auth/config"
	"github.com/gopherd/gopherd/auth/geo"
)

type testingService struct {
	config *config.Config
}

func (s *testingService) Config() *config.Config { return s.config }

func TestQueryLocation(t *testing.T) {
	service := new(testingService)
	service.config = (*config.Config)(nil).Default().(*config.Config)
	mod := geo.New(service)
	if err := mod.Init(); err != nil {
		t.Fatalf(err.Error())
	}
	t.Cleanup(mod.Shutdown)
	for _, tc := range [][2]string{
		{"en", "54.199.163.96"},
		{"zh-CN", "111.192.98.171"},
		{"en", "218.88.223.255"},
		{"zh-CN", "218.88.223.255"},
	} {
		country, province, city, err := mod.QueryLocation(tc[1], tc[0])
		if err != nil {
			t.Fatalf(err.Error())
		}
		t.Logf("country=%s, province=%s, city=%s", country, province, city)
	}
}
