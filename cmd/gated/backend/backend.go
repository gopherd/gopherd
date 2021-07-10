package backend

import (
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/jwt"
)

// Module used to connects backend servers
type Module interface {
	Busy() bool
	Forward(uid int64, typ proto.Type, body []byte) error
	Login(payload jwt.Payload, replace bool) error
	Logout(uid int64) error
}
