package backendmod

import (
	"strconv"

	"github.com/gopherd/doge/mq"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/service"
	"github.com/gopherd/doge/service/module"
	"github.com/gopherd/jwt"

	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/frontend"
	"github.com/gopherd/gopherd/proto/gatepb"
)

// New returns a backend module
func New(service Service) module.Module {
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

func (b *backendModule) Init() error {
	if err := b.BaseModule.Init(); err != nil {
		return err
	}
	topic := b.service.Name() + "/" + strconv.FormatInt(b.service.ID(), 10)
	b.service.MQ().Subscribe(topic, mq.FuncConsumer(b.consume))
	return nil
}

func (b *backendModule) consume(topic string, msg []byte, err error) {
	if err != nil {
		b.Logger().Warn().
			Error("error", err).
			Print("mq consume error")
		return
	}
	n, m, err := proto.Decode(msg)
	if err != nil {
		b.Logger().Error().
			Int("size", len(msg)).
			Error("error", err).
			Print("unmarshal mq message error")
		return
	}
	b.Logger().Debug().
		Int("size", len(msg)).
		Int("read", n).
		Int("type", int(m.Type())).
		Print("received a message from mq")
	switch ptc := m.(type) {
	case *gatepb.Broadcast:
		b.onBroadcast(ptc)
	case *gatepb.Response:
		b.onResponse(ptc)
	case *gatepb.Ping:
		b.onPing(ptc)
	case *gatepb.Pong:
		b.onPong(ptc)
	default:
		b.Logger().Warn().
			Int("size", len(msg)).
			Int("type", int(m.Type())).
			String("name", proto.Nameof(m)).
			Print("received a unknown message from mq")
	}
}

func (b *backendModule) onBroadcast(ptc *gatepb.Broadcast) {
	if len(ptc.Uids) == 0 {
		b.service.Frontend().BroadcastAll(ptc.Content)
	} else {
		b.service.Frontend().Broadcast(ptc.Uids, ptc.Content)
	}
}

func (b *backendModule) onResponse(ptc *gatepb.Response) {
	b.service.Frontend().Send(ptc.Uid, ptc.Content)
}

func (b *backendModule) onPing(ptc *gatepb.Ping) {
}

func (b *backendModule) onPong(ptc *gatepb.Pong) {
}

// Forward forwards message from frontend to backend
func (b *backendModule) Forward(uid int64, typ proto.Type, body proto.Body) error {
	return nil
}

func (b *backendModule) Login(uid int64, claims *jwt.Claims, userdata []byte) error {
	return nil
}

func (b *backendModule) Logout(uid int64) error {
	return nil
}
