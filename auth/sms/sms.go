package sms

import (
	"time"

	"github.com/gopherd/doge/service/module"
	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/config"
)

type Service interface {
	Config() *config.Config
}

func New(service Service) interface {
	module.Module
	auth.SMSModule
} {
	return newSMSModule(service)
}

// smsModule implements auth.SMSModule
type smsModule struct {
	*module.BaseModule
	service Service
}

func newSMSModule(service Service) *smsModule {
	return &smsModule{
		BaseModule: module.NewBaseModule("sms"),
		service:    service,
	}
}

func (mod *smsModule) GenerateCode(channel int, ip, mobile string) (time.Duration, error) {
	panic("TODO")
}
