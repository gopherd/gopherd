package provider

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrProviderNotFound = errors.New("auth: provider not found")
)

// error code
const (
	UnsupportedAPI      = "unsupported_api"
	NetworkError        = "network_error"
	ResponseFormatError = "response_format_error"
)

type Error struct {
	Name        string
	Code        string
	Description string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.Name, e.Code, e.Description)
}

// Gender of user
type Gender int

const (
	Unknown Gender = iota
	Male
	Female
)

// builtin provider
const Device = "device"

func Location(country, province, city string) string {
	if country != "" {
		if province != "" && province != country {
			if city != "" && city != province {
				return country + "," + province + "," + city
			}
			return country + "," + province
		}
		if city != "" && city != country {
			return country + "," + city
		}
		return country
	}
	if province != "" {
		if city != "" && city != province {
			return province + "," + city
		}
		return province
	}
	return city
}

type UserInfo struct {
	Key      string
	Name     string
	Avatar   string
	Gender   Gender
	Location string

	OpenId string
}

type Provider interface {
	Authorize(account, credentials string) (*UserInfo, error)
	Close() error
}

type Driver func(source string) (Provider, error)

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
	return driver(source)
}
