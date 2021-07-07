package backendmod

import (
	"strconv"

	"github.com/gopherd/doge/mq"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/service"
	"github.com/gopherd/doge/service/module"
	"github.com/gopherd/jwt"

	"github.com/gopherd/gopherd/cmd/gated/backend"
	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/frontend"
	"github.com/gopherd/gopherd/proto/gatepb"
)

// New returns a backend module
func New(service Service) interface {
	module.Module
	backend.Module
} {
	return newBackendModule(service)
}

// Service is required by backend module
type Service interface {
	service.Meta
	Config() *config.Config    // Config of service
	MQ() mq.Conn               // MQ instance
	Frontend() frontend.Module // Frontend module
}

// backendModule implements backend.Module interface
type backendModule struct {
	*module.BaseModule
	service Service
}

func newBackendModule(service Service) *backendModule {
	return &backendModule{
		BaseModule: module.NewBaseModule("backend"),
		service:    service,
	}
}

// Init overrides BaseModule Init method
func (mod *backendModule) Init() error {
	if err := mod.BaseModule.Init(); err != nil {
		return err
	}
	topic := mod.service.Name() + "/" + strconv.FormatInt(mod.service.ID(), 10)
	mod.service.MQ().Subscribe(topic, mq.FuncConsumer(mod.consume))
	return nil
}

// Busy implements backend.Module Busy method
func (mod *backendModule) Busy() bool {
	return false
}

// consume consumes message from mq
func (mod *backendModule) consume(topic string, msg []byte, err error) {
	if err != nil {
		mod.Logger().Warn().
			Error("error", err).
			Print("mq consume error")
		return
	}
	n, m, err := proto.Decode(msg)
	if err != nil {
		mod.Logger().Error().
			Int("size", len(msg)).
			Error("error", err).
			Print("unmarshal mq message error")
		return
	}
	mod.Logger().Debug().
		Int("size", len(msg)).
		Int("read", n).
		Int("type", int(m.Type())).
		Print("received a message from mq")
	switch ptc := m.(type) {
	case *gatepb.Broadcast:
		err = mod.broadcast(ptc)
	case *gatepb.Unicast:
		err = mod.unicast(ptc)
	case *gatepb.Kickout:
		err = mod.kickout(ptc)
	default:
		mod.Logger().Warn().
			Int("size", len(msg)).
			Int("type", int(m.Type())).
			String("name", proto.Nameof(m)).
			Print("received a unknown message from mq")
		return
	}
	if err != nil {
		mod.Logger().Warn().
			Int("type", int(m.Type())).
			String("name", proto.Nameof(m)).
			Error("error", err).
			Print("handle message error")
	}
}

// broadcast handles Broadcast message
func (mod *backendModule) broadcast(ptc *gatepb.Broadcast) error {
	if len(ptc.Uids) == 0 {
		return mod.service.Frontend().BroadcastAll(ptc.Content)
	} else {
		return mod.service.Frontend().Broadcast(ptc.Uids, ptc.Content)
	}
}

// unicast handles Unicast message
func (mod *backendModule) unicast(ptc *gatepb.Unicast) error {
	return mod.service.Frontend().Unicast(ptc.Uid, ptc.Content)
}

// kickout handles Kickout message
func (mod *backendModule) kickout(ptc *gatepb.Kickout) error {
	return mod.service.Frontend().Kickout(ptc.Uid, gatepb.KickoutReason(ptc.Reason))
}

// Forward implements backend.Module Forward method
func (mod *backendModule) Forward(uid int64, typ proto.Type, body proto.Body) error {
	// (TODO) where to forward
	return nil
}

// Login implements backend.Module Login method
func (mod *backendModule) Login(claims jwt.Payload, userdata []byte, replace bool) error {
	// (TODO) handle login
	return nil
}

// Logout implements backend.Module Logout method
func (mod *backendModule) Logout(uid int64) error {
	// (TODO) handle logout
	return nil
}
