package auth

import (
	"net/http"
	"time"

	"github.com/gopherd/gopherd/auth/provider"
	"github.com/gopherd/jwt"
	"github.com/gopherd/log"
)

type Account interface {
	GetID() int64
	SetID(id int64)
	GetDeviceID() string
	SetDeviceID(string)
	GetBanned() (bool, string)
	SetBanned(bool, string)
	GetRegister() (time.Time, string)
	SetRegister(time.Time, string)
	GetLastLogin() (time.Time, string)
	SetLastLogin(time.Time, string)
	GetName() string
	SetName(string)
	GetAvatar() string
	SetAvatar(string)
	GetGender() int
	SetGender(int)
	GetLocation() string
	SetLocation(string)
	GetProvider(string) string
	SetProvider(provider, key string)
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
	Signer() *jwt.Signer
	Provider(name string) (provider.Provider, error)
	Response(w http.ResponseWriter, r *http.Request, v interface{}) error
	GenerateSMSCode(channel int, ip, mobile string) (time.Duration, error)
	AccountManager() AccountManager
	QueryLocationByIP(ip string) string
}

type AccountManager interface {
	Exist(provider, key string) (bool, error)
	Store(provider string, account Account) error
	Load(provider, key string) (Account, error)
	LoadOrCreate(provider, key, device string) (Account, bool, error)
	Get(uid int64) (Account, bool, error)
}
