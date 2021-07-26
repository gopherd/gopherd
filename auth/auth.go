package auth

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gopherd/gopherd/auth/api"
	"github.com/gopherd/jwt"
	"github.com/gopherd/log"
)

type Account struct {
	Uid          int64     `json:"uid"`
	OpenId       string    `json:"open_id"`
	Banned       bool      `json:"banned"`
	BannedReason string    `json:"banned_reason"`
	LastLoginAt  time.Time `json:"last_login_at"`
	LastLoginIP  string    `json:"last_login_ip"`
}

type Options struct {
	JWT struct {
		Filename string `json:"filename"`
		Issuer   string `json:"issuer"`
		KeyId    string `json:"key_id"`
	} `json:"jwt"`
	AccessTokenTTL  int64 `json:"access_token_ttl"`
	RefreshTokenTTL int64 `json:"refresh_token_ttl"`
}

type Handler func(Service, http.ResponseWriter, *http.Request)

type Service interface {
	Options() Options
	Logger() *log.Logger
	Provider(name string) (Provider, error)
	Response(w http.ResponseWriter, r *http.Request, v interface{}) error
	Signer() *jwt.Signer
	GenerateSMSCode(channel int, ip, mobile string) (time.Duration, error)
}

type Provider interface {
	Authorize(ip string, req *api.AuthorizeRequest) (account *Account, isNew bool, err error)
	Link(ip string, req *api.LinkRequest, claims *jwt.Claims) (account *Account, err error)
}

type Driver interface {
	Open(source string) (Provider, error)
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

func Register(name string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if _, dup := drivers[name]; dup {
		panic("auth: Register " + name + " called twice")
	}
	drivers[name] = driver
}

func Open(name string, source string) (Provider, error) {
	driver, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf("auth: provider %q not found, forgot import?", name)
	}
	return driver.Open(source)
}
