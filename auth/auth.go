package auth

import (
	"time"

	"github.com/gopherd/gopherd/auth/provider"
	"github.com/gopherd/jwt"
	"github.com/gopherd/log"
)

type Account interface {
	GetID() int64
	SetID(int64)
	GetDeviceID() string
	SetDeviceID(string)
	GetBanned() (bool, string)
	SetBanned(bool, string)
	GetRegister() (time.Time, string)
	SetRegister(at time.Time, ip string)
	GetLastLogin() (time.Time, string)
	SetLastLogin(at time.Time, ip string)
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
	Routers         struct {
		Authorize string `json:"authorize"` // default: /auth/authorize
		Link      string `json:"link"`      // default: /auth/link
		SMSCode   string `json:"smscode"`   // default: /auth/smscode
	} `json:"routers"`
}

type Service interface {
	Options() *Options
	Logger() *log.Logger
	Signer() *jwt.Signer
	Provider(name string) (provider.Provider, error)
	OOSModule() OOSModule
	AccountModule() AccountModule
	SMSModule() SMSModule
	GeoModule() GeoModule
}

// OOSModule reprensets an object-oriented storage system
type OOSModule interface {
	GetObject(obj interface{}, where ...Cond) (bool, error)
	HasObject(key string, where ...Cond) (bool, error)
	InsertObject(obj interface{}) error
	UpdateObject(obj interface{}, fields ...string) (int64, error)
}

type Cond struct {
	Field string
	Value string
}

type AccountModule interface {
	Exist(provider, key string) (bool, error)
	Store(provider string, account Account) error
	Load(provider, key string) (Account, error)
	LoadOrCreate(provider, key, device string) (Account, bool, error)
	Get(uid int64) (Account, error)
}

type SMSModule interface {
	GenerateCode(channel int, ip, mobile string) (time.Duration, error)
}

type GeoModule interface {
	QueryLocationByIP(ip string) string
}
