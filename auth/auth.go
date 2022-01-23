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
	GetProviders() map[string]string
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
	UpdateObject(obj Object, fields ...any) (int64, error)
}

type Field struct {
	Name  string
	Value string
}

const FieldId = "id"

func ByProvider(name, key string) Field {
	return Field{
		Name:  provider.ProviderFieldName(name),
		Value: key,
	}
}

func ByID(id int64) Field {
	return Field{
		Name:  FieldId,
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
