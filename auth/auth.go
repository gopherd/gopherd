package auth

import (
	"strconv"
	"time"

	"github.com/gopherd/gopherd/auth/config"
	"github.com/gopherd/gopherd/auth/provider"
	"github.com/gopherd/jwt"
	"github.com/gopherd/log"
)

type Object interface {
	TableName() string
}

type Providers interface {
	Get(name string) string
	Set(name, value string)
}

var NewProviders func() Providers = newProviders

// default DefaultProviders
type DefaultProviders struct {
	Mobile   string `gorm:"uniqueIndex"`
	Email    string `gorm:"uniqueIndex"`
	Google   string `gorm:"uniqueIndex"`
	Line     string `gorm:"uniqueIndex"`
	Facebook string `gorm:"uniqueIndex"`
	Wechat   string `gorm:"uniqueIndex"`
	Toutiao  string `gorm:"uniqueIndex"`
}

func newProviders() Providers { return new(DefaultProviders) }

func (p *DefaultProviders) Get(name string) string {
	switch name {
	case "mobile":
		return p.Mobile
	case "email":
		return p.Email
	case "google":
		return p.Google
	case "line":
		return p.Line
	case "facebook":
		return p.Facebook
	case "wechat", "wxgame":
		return p.Wechat
	case "toutiao", "ttgame":
		return p.Toutiao
	default:
		return ""
	}
}

func (p *DefaultProviders) Set(name, value string) {
	switch name {
	case "mobile":
		p.Mobile = value
	case "email":
		p.Email = value
	case "google":
		p.Google = value
	case "line":
		p.Line = value
	case "facebook":
		p.Facebook = value
	case "wechat", "wxgame":
		p.Wechat = value
	case "toutiao", "ttgame":
		p.Toutiao = value
	default:
	}
}

type Account interface {
	Object
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
	GetProviders() Providers
}

type Service interface {
	Config() *config.Config
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
	CreateSchema(Object) error
	GetObject(obj Object, by ...Field) (bool, error)
	HasObject(tableName string, by ...Field) (bool, error)
	InsertObject(obj Object) error
	UpdateObject(obj Object, fields ...string) (int64, error)
}

type Field struct {
	Name  string
	Value string
}

func ByProvider(name, key string) Field {
	return Field{
		Name:  "provider_" + name,
		Value: key,
	}
}

func ByID(id int64) Field {
	return Field{
		Name:  "id",
		Value: strconv.FormatInt(id, 10),
	}
}

type AccountModule interface {
	Contains(by ...Field) (bool, error)
	Store(provider string, account Account) error
	Load(by ...Field) (Account, error)
	LoadOrCreate(provider, key, device string) (Account, bool, error)
}

type SMSModule interface {
	GenerateCode(channel int, ip, mobile string) (time.Duration, error)
}

type GeoModule interface {
	QueryLocation(ip, lang string) (country, province, city string, err error)
}
