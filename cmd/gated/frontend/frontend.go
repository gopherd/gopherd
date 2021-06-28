package frontend

import "github.com/gopherd/gopherd/proto/gatepb"

const UsersTable = "gated/users"

// Module managers client sessions
type Module interface {
	Broadcast(uids []int64, content []byte) error
	BroadcastAll(content []byte) error
	Send(uid int64, content []byte) error
	Kickout(uid int64, reason gatepb.KickoutReason) error
}
