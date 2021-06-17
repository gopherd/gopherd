package module

import (
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/gopherd/proto/gatepb"
)

// Frontend managers client sessions
type Frontend interface {
	Broadcast(uids []int64, content []byte) error
	BroadcastAll(content []byte) error
	Send(uid int64, content []byte) error
	Kickout(uid int64, reason gatepb.KickoutReason) error
}

// Backend connects to backend servers
type Backend interface {
	Forward(uid int64, typ proto.Type, body proto.Body) error
	Login(uid int64, req *gatepb.Login) error
	Logout(uid int64, req *gatepb.Logout) error
}
