package module

import (
	"github.com/gopherd/doge/proto"
)

type Frontend interface {
	Broadcast(content []byte) error
	BroadcastTo(uids []int64, content []byte) error
}

type Backend interface {
	Forward(typ proto.Type, body proto.Body) error
}
