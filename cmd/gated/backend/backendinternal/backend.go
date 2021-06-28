package backendinternal

import (
	"strconv"

	"github.com/gopherd/doge/mq"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/service"
	"github.com/gopherd/doge/service/component"
	"github.com/gopherd/jwt"

	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/frontend"
	"github.com/gopherd/gopherd/proto/gatepb"
)

// New returns a backend component
func New(service Service) component.Component {
	return newBackendComponent(service)
}

// Service is required by backend component
type Service interface {
	service.Meta
	Config() *config.Config       // Config of service
	MQ() mq.Conn                  // MQ instance
	Frontend() frontend.Component // Frontend component
}

// backendComponent implements backend.Component interface
type backendComponent struct {
	*component.BaseComponent
	service Service
}

func newBackendComponent(service Service) *backendComponent {
	return &backendComponent{
		BaseComponent: component.NewBaseComponent("backend"),
		service:       service,
	}
}

func (b *backendComponent) Init() error {
	if err := b.BaseComponent.Init(); err != nil {
		return err
	}
	topic := b.service.Name() + "/" + strconv.FormatInt(b.service.ID(), 10)
	b.service.MQ().Subscribe(topic, mq.FuncConsumer(b.consume))
	return nil
}

func (b *backendComponent) consume(topic string, msg []byte, err error) {
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

func (b *backendComponent) onBroadcast(ptc *gatepb.Broadcast) {
	if len(ptc.Uids) == 0 {
		b.service.Frontend().BroadcastAll(ptc.Content)
	} else {
		b.service.Frontend().Broadcast(ptc.Uids, ptc.Content)
	}
}

func (b *backendComponent) onResponse(ptc *gatepb.Response) {
	b.service.Frontend().Send(ptc.Uid, ptc.Content)
}

func (b *backendComponent) onPing(ptc *gatepb.Ping) {
}

func (b *backendComponent) onPong(ptc *gatepb.Pong) {
}

// Forward forwards message from frontend to backend
func (b *backendComponent) Forward(uid int64, typ proto.Type, body proto.Body) error {
	return nil
}

func (b *backendComponent) Login(uid int64, claims *jwt.Claims, userdata []byte) error {
	return nil
}

func (b *backendComponent) Logout(uid int64) error {
	return nil
}
