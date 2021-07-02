package frontend

import (
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/gopherd/proto/gatepb"
)

const UsersTable = "gated/users"

// Module managers client sessions
type Module interface {
	Busy() bool
	Broadcast(uids []int64, content []byte) error
	BroadcastAll(content []byte) error
	Write(uid int64, content []byte) error
	Send(uid int64, m proto.Message) error
	Kickout(uid int64, reason gatepb.KickoutReason) error
}
