package sms

import (
	"time"

	"github.com/gopherd/doge/service/module"
	"github.com/gopherd/gopherd/auth"
)

func New(service auth.Service) interface {
	module.Module
	auth.SMSModule
} {
	return newSMSModule(service)
}

// smsModule implements auth.SMSModule
type smsModule struct {
	module.BaseModule
	service auth.Service
}

func newSMSModule(service auth.Service) *smsModule {
	return &smsModule{
		service: service,
	}
}

func (mod *smsModule) GenerateCode(channel int, ip, mobile string) (time.Duration, error) {
	panic("TODO")
}
