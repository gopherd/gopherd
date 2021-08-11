package backendmod

import (
	"encoding/json"
	"errors"
	"net"
	"path"
	"strconv"
	"sync"

	"github.com/gopherd/doge/mq"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/proto/router"
	"github.com/gopherd/doge/service"
	"github.com/gopherd/doge/service/discovery"
	"github.com/gopherd/doge/service/module"
	"github.com/gopherd/jwt"

	"github.com/gopherd/gopherd/gate/backend"
	"github.com/gopherd/gopherd/gate/config"
	"github.com/gopherd/gopherd/gate/frontend"
	"github.com/gopherd/gopherd/proto/gatepb"
)

var errUnknownMessage = errors.New("backend: unknown message")
var (
	forwardPool = sync.Pool{
		New: func() interface{} {
			return new(gatepb.Forward)
		},
	}
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
	Config() *config.Config         // Config of service
	MQ() mq.Conn                    // MQ instance
	Discovery() discovery.Discovery // Discovery instance
	Frontend() frontend.Module      // Frontend module
}

// backendModule implements backend.Module interface
type backendModule struct {
	*module.BaseModule
	service Service
	routers *router.Cache
	arena   proto.Arena
}

func newBackendModule(service Service) *backendModule {
	return &backendModule{
		BaseModule: module.NewBaseModule("backend"),
		service:    service,
		arena:      new(arena),
	}
}

// Init overrides BaseModule Init method
func (mod *backendModule) Init() error {
	if err := mod.BaseModule.Init(); err != nil {
		return err
	}
	mod.routers = router.NewCache(mod.service.Discovery())
	if err := mod.routers.Init(); err != nil {
		return err
	}
	topic := path.Join(mod.service.Name(), strconv.FormatInt(mod.service.ID(), 10))
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
	n, m, err := proto.Decode(msg, mod.arena)
	if err != nil {
		mod.Logger().Error().
			Int("size", len(msg)).
			Error("error", err).
			Print("unmarshal message from mq error")
		return
	}
	defer mod.arena.Put(m)
	mod.Logger().Debug().
		Int("size", len(msg)).
		Int("read", n).
		Int("type", int(m.Typeof())).
		Print("received a message from mq")

	switch ptc := m.(type) {
	case *gatepb.Unicast:
		err = mod.service.Frontend().Unicast(ptc.Uid, ptc.Msg)
	case *gatepb.Multicast:
		err = mod.service.Frontend().Multicast(ptc.Uids, ptc.Msg)
	case *gatepb.Broadcast:
		err = mod.service.Frontend().Broadcast(ptc.Msg)
	case *gatepb.Kickout:
		err = mod.service.Frontend().Kickout(ptc.Uid, gatepb.KickoutReason(ptc.Reason))
	case *gatepb.Router:
		if ptc.Addr == "" {
			mod.routers.Remove(ptc.Mod)
		} else {
			mod.routers.Add(ptc.Mod, ptc.Addr)
		}
	default:
		err = errUnknownMessage
	}

	if err != nil {
		mod.Logger().Warn().
			Int("type", int(m.Typeof())).
			String("name", m.Nameof()).
			Error("error", err).
			Print("handle message error")
	}
}

// Forward implements backend.Module Forward method
func (mod *backendModule) Forward(f *gatepb.Forward) error {
	return mod.send(proto.Type(f.Typ), f)
}

// Login implements backend.Module Login method
func (mod *backendModule) Login(claims jwt.Payload, race bool) error {
	userdata, err := json.Marshal(claims.Values)
	if err != nil {
		return err
	}
	m := &gatepb.Login{
		Gid:      int64(mod.service.ID()),
		Uid:      claims.ID,
		Ip:       []byte(net.ParseIP(claims.IP)),
		Userdata: userdata,
		Race:     race,
	}
	return mod.send(m.Typeof(), m)
}

// Logout implements backend.Module Logout method
func (mod *backendModule) Logout(uid int64) error {
	m := &gatepb.Logout{
		Uid: uid,
	}
	return mod.send(m.Typeof(), m)
}

// send message to mq.
// typ maybe not equal to m.Typeof()
func (mod *backendModule) send(typ proto.Type, m proto.Message) error {
	modName := proto.Moduleof(typ)
	if modName == "" {
		mod.Logger().Warn().
			Int("type", int(typ)).
			Print("module not found")
		return proto.ErrUnrecognizedType(typ)
	}
	topic, err := mod.routers.Lookup(modName)
	if err != nil {
		mod.Logger().Warn().
			Int("type", int(typ)).
			String("module", modName).
			Print("router not found")
		return err
	}
	buf := proto.AllocBuffer()
	defer proto.FreeBuffer(buf)
	if err := buf.Marshal(m); err != nil {
		mod.Logger().Warn().
			String("topic", topic).
			Int("type", int(typ)).
			Print("marshal message error")
		return err
	}
	return mod.service.MQ().Publish(topic, buf.Bytes())
}
