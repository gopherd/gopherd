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
	module.BaseModule
	service Service
}

func newAccountModule(service Service) *accountModule {
	return &accountModule{
		service: service,
	}
}

func (mod *accountModule) Contains(by ...auth.Field) (bool, error) {
	return mod.service.OOSModule().HasObject(tableName)
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
		return nil, false, nil
	}
	return a, true, nil
}
