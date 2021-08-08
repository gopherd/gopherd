package account

import (
	"github.com/gopherd/doge/service/module"

	"github.com/gopherd/gopherd/auth"
)

// New creates an auth.AccountModule
func New(service auth.Service) interface {
	module.Module
	auth.AccountModule
} {
	return newAccountModule(service)
}

// accountModule implements auth.AccountModule
type accountModule struct {
	module.BaseModule
	service auth.Service
}

func newAccountModule(service auth.Service) *accountModule {
	return &accountModule{
		service: service,
	}
}

func (mod *accountModule) Exist(provider, key string) (bool, error) {
	return mod.service.OOSModule().HasObject(tableName, byProvider(provider, key))
}

func (mod *accountModule) Store(provider string, account auth.Account) error {
	_, err := mod.service.OOSModule().UpdateObject(account)
	return err
}

func (mod *accountModule) Load(provider, key string) (auth.Account, error) {
	a := newAccount()
	found, err := mod.service.OOSModule().GetObject(a, byProvider(provider, key))
	if err != nil {
		return nil, err
	} else if !found {
		return nil, err
	}
	return a, nil
}

func (mod *accountModule) LoadOrCreate(provider, key, device string) (auth.Account, bool, error) {
	a := newAccount()
	found, err := mod.service.OOSModule().GetObject(a, byProvider(provider, key))
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

func (mod *accountModule) Get(id int64) (auth.Account, error) {
	a := newAccount()
	found, err := mod.service.OOSModule().GetObject(a, byID(id))
	if err != nil {
		return nil, err
	} else if !found {
		return nil, err
	}
	return a, nil
}
