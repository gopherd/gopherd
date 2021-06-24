package internal

import (
	"strconv"

	"github.com/gopherd/doge/mq"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/service/component"
	"github.com/gopherd/jwt"

	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/module"
	"github.com/gopherd/gopherd/proto/gatepb"
)

type Service interface {
	Config() *config.Config
	MQ() mq.Conn
	ID() int64
	Name() string
	Frontend() module.Frontend
}

func NewComponent(service Service) *backend {
	return newBackend(service)
}

type backend struct {
	*component.BaseComponent
	service Service
}

func newBackend(service Service) *backend {
	return &backend{
		BaseComponent: component.NewBaseComponent("backend"),
		service:       service,
	}
}

func (b *backend) Init() error {
	if err := b.BaseComponent.Init(); err != nil {
		return err
	}
	topic := b.service.Name() + "/" + strconv.FormatInt(b.service.ID(), 10)
	b.service.MQ().Subscribe(topic, mq.FuncConsumer(b.consume))
	return nil
}

func (b *backend) consume(topic string, msg []byte, err error) {
	if err != nil {
		b.Logger().Warn().
			Error("error", err).
			Print("mq consume error")
		return
	}
	m, err := proto.DecodeBody(msg)
	if err != nil {
		b.Logger().Error().
			Int("size", len(msg)).
			Error("error", err).
			Print("unmarshal mq message error")
		return
	}
	b.Logger().Debug().
		Int("size", len(msg)).
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

func (b *backend) onBroadcast(ptc *gatepb.Broadcast) {
	if len(ptc.Uids) == 0 {
		b.service.Frontend().BroadcastAll(ptc.Content)
	} else {
		b.service.Frontend().Broadcast(ptc.Uids, ptc.Content)
	}
}

func (b *backend) onResponse(ptc *gatepb.Response) {
	b.service.Frontend().Send(ptc.Uid, ptc.Content)
}

func (b *backend) onPing(ptc *gatepb.Ping) {
}

func (b *backend) onPong(ptc *gatepb.Pong) {
}

// Forward forwards message from frontend to backend
func (b *backend) Forward(uid int64, typ proto.Type, body proto.Body) error {
	return nil
}

func (b *backend) Login(uid int64, claims *jwt.Claims, userdata []byte) error {
	return nil
}

func (b *backend) Logout(uid int64) error {
	return nil
}
