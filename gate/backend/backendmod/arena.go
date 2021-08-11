package backendmod

import (
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/gopherd/proto/gatepb"
)

type arena struct {
	pool proto.Pool
}

func (arena *arena) Get(typ proto.Type) proto.Message {
	switch typ {
	case gatepb.UnicastType,
		gatepb.MulticastType,
		gatepb.BroadcastType,
		gatepb.KickoutType,
		gatepb.RouterType:
	default:
		return nil
	}
	return arena.pool.Get(typ)
}

func (arena *arena) Put(m proto.Message) {
	arena.pool.Put(m)
}
