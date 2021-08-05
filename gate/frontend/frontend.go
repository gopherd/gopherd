package frontend

import (
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/gopherd/proto/gatepb"
)

const UsersTable = "gated/users"

// Module managers client sessions
type Module interface {
	Busy() bool
	Unicast(uid int64, content []byte) error
	Multicast(uids []int64, content []byte) error
	Broadcast(content []byte) error
	Send(uid int64, m proto.Message) error
	Kickout(uid int64, reason gatepb.KickoutReason) error
}
