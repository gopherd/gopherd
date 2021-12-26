package account

import (
	"github.com/gopherd/doge/service/module"

	"github.com/gopherd/gopherd/auth"
)

type Service interface {
	OOSModule() auth.OOSModule
}

// New creates an auth.AccountModule
func New(service Service) interface {
	module.Module
	auth.AccountModule
} {
	return newAccountModule(service)
}

// accountModule implements auth.AccountModule
type accountModule struct {
	*module.BaseModule
	service Service
}

func newAccountModule(service Service) *accountModule {
	return &accountModule{
		BaseModule: module.NewBaseModule("account"),
		service:    service,
	}
}

func (mod *accountModule) Init() error {
	if err := mod.BaseModule.Init(); err != nil {
		return err
	}
	return mod.service.OOSModule().CreateSchema(newAccount())
}

func (mod *accountModule) Contains(by ...auth.Field) (bool, error) {
	return mod.service.OOSModule().HasObject(tableName, by...)
}

func (mod *accountModule) Store(provider string, account auth.Account) error {
	_, err := mod.service.OOSModule().UpdateObject(account)
	return err
}

func (mod *accountModule) Load(by ...auth.Field) (auth.Account, error) {
	a := newAccount()
	found, err := mod.service.OOSModule().GetObject(a, by...)
	if err != nil {
		return nil, err
	} else if !found {
		return nil, err
	}
	return a, nil
}

// FIXME: 当 provider != deviceid 时需要检查改 provider 是否被其他的用户使用
func (mod *accountModule) LoadOrCreate(provider, key, device string) (auth.Account, bool, error) {
	a := newAccount()
	found, err := mod.service.OOSModule().GetObject(a, auth.ByProvider(provider, key))
	if err != nil {
		return nil, false, err
	} else if found {
		return a, false, nil
	}
	a.DeviceID = device
	a.SetProvider(provider, key)
	if err := mod.service.OOSModule().InsertObject(a); err != nil {
		return nil, false, err
	}
	return a, true, nil
}
