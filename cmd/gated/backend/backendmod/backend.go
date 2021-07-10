package backendmod

import (
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

	"github.com/gopherd/gopherd/cmd/gated/backend"
	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/frontend"
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
	service     Service
	routerCache *router.Cache
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
	mod.routerCache = router.NewCache(mod.service.Discovery())
	if err := mod.routerCache.Init(); err != nil {
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
	n, m, err := proto.Decode(msg)
	if err != nil {
		mod.Logger().Error().
			Int("size", len(msg)).
			Error("error", err).
			Print("unmarshal message from mq error")
		return
	}
	mod.Logger().Debug().
		Int("size", len(msg)).
		Int("read", n).
		Int("type", int(m.Type())).
		Print("received a message from mq")

	switch ptc := m.(type) {
	case *gatepb.RegisterRouter:
		mod.routerCache.Add(ptc.Mod, ptc.Addr)
	case *gatepb.UnregisterRouter:
		mod.routerCache.Remove(ptc.Mod)
	case *gatepb.Broadcast:
		if len(ptc.Uids) == 0 {
			err = mod.service.Frontend().BroadcastAll(ptc.Content)
		} else {
			err = mod.service.Frontend().Broadcast(ptc.Uids, ptc.Content)
		}
	case *gatepb.Unicast:
		err = mod.service.Frontend().Unicast(ptc.Uid, ptc.Content)
	case *gatepb.Kickout:
		err = mod.service.Frontend().Kickout(ptc.Uid, gatepb.KickoutReason(ptc.Reason))
	default:
		err = errUnknownMessage
	}

	if err != nil {
		mod.Logger().Warn().
			Int("type", int(m.Type())).
			String("name", proto.Nameof(m)).
			Error("error", err).
			Print("handle message error")
	}
}

// Forward implements backend.Module Forward method
func (mod *backendModule) Forward(uid int64, typ proto.Type, body []byte) error {
	m := forwardPool.Get().(*gatepb.Forward)
	m.Reset()
	m.Gid = int64(mod.service.ID())
	m.Uid = uid
	m.MsgType = int32(typ)
	m.MsgContent = []byte(body)
	if len(m.MsgContent) < (1 << 12) {
		defer forwardPool.Put(m)
	}
	return mod.send(typ, m)
}

// Login implements backend.Module Login method
func (mod *backendModule) Login(claims jwt.Payload, replace bool) error {
	m := &gatepb.UserLogin{
		Gid:      int64(mod.service.ID()),
		Uid:      claims.ID,
		Ip:       []byte(net.ParseIP(claims.IP)),
		Userdata: []byte(claims.Userdata),
		Replace:  replace,
	}
	return mod.send(m.Type(), m)
}

// Logout implements backend.Module Logout method
func (mod *backendModule) Logout(uid int64) error {
	m := &gatepb.UserLogout{
		Uid: uid,
	}
	return mod.send(m.Type(), m)
}

// send message to mq.
// typ maybe not equal to m.Type()
func (mod *backendModule) send(typ proto.Type, m proto.Message) error {
	modName := proto.Moduleof(typ)
	if modName == "" {
		mod.Logger().Warn().
			Int("type", int(typ)).
			Print("module not found")
		return proto.ErrUnrecognizedType
	}
	topic, err := mod.routerCache.Lookup(modName)
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
