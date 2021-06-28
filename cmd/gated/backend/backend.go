package backend

import (
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/jwt"
)

// Component used to connects backend servers
type Component interface {
	Forward(uid int64, typ proto.Type, body proto.Body) error
	Login(uid int64, claims *jwt.Claims, userdata []byte) error
	Logout(uid int64) error
}
