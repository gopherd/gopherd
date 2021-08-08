package backend

import (
	"github.com/gopherd/gopherd/proto/gatepb"
	"github.com/gopherd/jwt"
)

// Module used to connects backend servers
type Module interface {
	Busy() bool
	Forward(*gatepb.Forward) error
	Login(payload jwt.Payload, race bool) error
	Logout(uid int64) error
}
