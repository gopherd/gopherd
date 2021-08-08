package frontend

import (
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/gopherd/proto/gatepb"
)

const UsersTable = "gated/users"

// Module managers client sessions
type Module interface {
	Busy() bool
	Unicast(uid int64, msg []byte) error
	Multicast(uids []int64, msg []byte) error
	Broadcast(msg []byte) error
	Send(uid int64, m proto.Message) error
	Kickout(uid int64, reason gatepb.KickoutReason) error
}
