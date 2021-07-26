package frontendmod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/doge/net/netutil"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/service"
	"github.com/gopherd/doge/service/discovery"
	"github.com/gopherd/doge/service/module"
	"github.com/gopherd/doge/text/resp"
	"github.com/gopherd/doge/time/timer"
	"github.com/gopherd/jwt"

	"github.com/gopherd/gopherd/gate/backend"
	"github.com/gopherd/gopherd/gate/config"
	"github.com/gopherd/gopherd/gate/frontend"
	"github.com/gopherd/gopherd/proto/gatepb"
)

// New returns a frontend moudle
func New(service Service) interface {
	module.Module
	frontend.Module
} {
	return newFrontendModule(service)
}

// Service is required by frontend module
type Service interface {
	service.Meta
	Config() *config.Config         // Config of service
	Discovery() discovery.Discovery // Discovery instance
	Backend() backend.Module        // Backend module
}

// frontendModule implements frontend.Module interface
type frontendModule struct {
	*module.BaseModule
	service      Service
	shuttingDown int32

	verifier *jwt.Verifier
	server   interface{ Serve(net.Listener) error }
	listener net.Listener

	sessions              *sessions
	pendingSessions       sync.Map
	pendingSessionsTicker *timer.Ticker
}

func newFrontendModule(service Service) *frontendModule {
	mod := &frontendModule{
		BaseModule:            module.NewBaseModule("frontend"),
		service:               service,
		pendingSessionsTicker: timer.NewTicker(time.Second),
	}
	mod.sessions = newSessions(mod)
	return mod
}

// Init overrides BaseModule Init method
func (mod *frontendModule) Init() error {
	if err := mod.BaseModule.Init(); err != nil {
		return err
	}
	cfg := mod.service.Config()

	// create jwt verifier
	if verifier, err := jwt.NewVerifier(cfg.JWT.Filename, cfg.JWT.KeyId); err != nil {
		return erron.Throw(err)
	} else {
		mod.verifier = verifier
	}

	// init sessions
	mod.sessions.init()

	// start tcp/websocket server
	if cfg.Net.Port <= 0 {
		return erron.Throwf("invalid port: %d", cfg.Net.Port)
	}
	addr := fmt.Sprintf("%s:%d", cfg.Net.Bind, cfg.Net.Port)
	keepalive := time.Duration(cfg.Net.Keepalive) * time.Second
	if cfg.Net.Protocol == "tcp" {
		server, listener, err := netutil.ListenTCP(addr, mod.onOpen, keepalive)
		if err != nil {
			return erron.Throw(err)
		}
		mod.server = server
		mod.listener = listener
	} else {
		server, listener, err := httputil.ListenWebsocket(addr, "/", mod.onOpen, keepalive)
		if err != nil {
			return erron.Throw(err)
		}
		mod.server = server
		mod.listener = listener
	}
	mod.Logger().Info().
		String("protocol", cfg.Net.Protocol).
		String("addr", addr).
		Print("listening")

	return nil
}

// Start overrides BaseModule Start method
func (mod *frontendModule) Start() {
	mod.BaseModule.Start()
	go mod.server.Serve(mod.listener)
}

// Shutdown overrides BaseModule Shutdown method
func (mod *frontendModule) Shutdown() {
	mod.BaseModule.Shutdown()
	mod.clean()
}

func (mod *frontendModule) clean() {
	mod.sessions.shutdown()
}

// Update overrides BaseModule Update method
func (mod *frontendModule) Update(now time.Time, dt time.Duration) {
	mod.BaseModule.Update(now, dt)

	if mod.service.State() == service.Stopping {
		if atomic.CompareAndSwapInt32(&mod.shuttingDown, 0, 1) {
			mod.clean()
		}
	} else {
		if mod.pendingSessionsTicker.Next(now) {
			timestamp := now.UnixNano() / 1e6
			deleted := make(map[int64]bool)
			mod.pendingSessions.Range(func(k, v interface{}) bool {
				sid := k.(int64)
				ps := v.(*pendingSession)
				if mod.retryLogin(sid, ps, timestamp) {
					deleted[sid] = true
				}
				return true
			})
			for k := range deleted {
				mod.pendingSessions.Delete(k)
			}
		}
		mod.sessions.clean(now)
	}
}

// Busy implements frontend.Module Busy method
func (mod *frontendModule) Busy() bool {
	return mod.sessions.size() > 0
}

// onOpen implements handler onOpen method
func (mod *frontendModule) onOpen(ip string, conn net.Conn) {
	sid := mod.sessions.allocSessionId()
	mod.Logger().Debug().
		Int64("sid", sid).
		String("ip", ip).
		Print("session connected")
	s := newSession(mod.Logger(), sid, ip, conn, mod)
	// Blocked here
	s.serve()
}

// onReady implements handler onReady method
func (mod *frontendModule) onReady(s *session) {
	n, ok := mod.sessions.add(s)
	if !ok {
		mod.Logger().Warn().
			Int64("sid", s.id).
			Int("sessions", n).
			Print("add session failed")
	} else {
		mod.Logger().Debug().
			Int64("sid", s.id).
			Int("sessions", n).
			Print("session ready")
	}
}

// onClose implements handler onClose method
func (mod *frontendModule) onClose(s *session, err error) {
	mod.Logger().Debug().Int64("sid", s.id).Print("session closed")
	if s.getUid() > 0 {
		mod.afterLogout(s)
	} else {
		mod.sessions.remove(s.id)
	}
}

// onMessage implements handler onMessage method
func (mod *frontendModule) onMessage(s *session, typ proto.Type, body proto.Body) error {
	mod.Logger().Trace().
		Int64("sid", s.id).
		Int("size", body.Len()).
		Int("type", int(typ)).
		Print("session received a typed message")
	var (
		m   proto.Message
		err error
	)
	switch typ {
	case gatepb.PingType:
		err = mod.ping(s, typ, body)
	case gatepb.LoginReqType:
		if m, err = mod.unmarshal(s, typ, body); err == nil {
			err = mod.login(s, m.(*gatepb.LoginReq))
		}
	case gatepb.LogoutReqType:
		if m, err = mod.unmarshal(s, typ, body); err == nil {
			err = mod.logout(s, m.(*gatepb.LogoutReq))
		}
	default:
		err = mod.forward(s, typ, body)
	}
	if err != nil {
		mod.Logger().Warn().
			Int64("sid", s.id).
			Int("type", int(typ)).
			Error("error", err).
			Print("session handle message error")
	}
	return err
}

// onCommand implements handler onCommand method
func (mod *frontendModule) onCommand(s *session, cmd *resp.Command) error {
	name := cmd.Name()
	c := commands[strings.ToLower(name)]
	if c == nil {
		typ, err := proto.ParseType(name)
		if err != nil {
			if errors.Is(err, proto.ErrTypeOverflow) {
				return err
			}
			return errorln(s, "command "+name+" not found, run command to list all supported commands")
		}
		switch cmd.NArg() {
		case 0:
			return mod.onMessage(s, typ, proto.Text(nil))
		case 1:
			return mod.onMessage(s, typ, proto.Text([]byte(cmd.Arg(0))))
		default:
			return resp.ErrNumberOfArguments
		}
	}
	return c.run(mod, s, cmd)
}

func (mod *frontendModule) unmarshal(s *session, typ proto.Type, body proto.Body) (proto.Message, error) {
	m := proto.New(typ)
	if m == nil {
		return nil, proto.ErrUnrecognizedType
	}
	if size := body.Len(); size > 0 {
		buf := proto.AllocBuffer()
		defer proto.FreeBuffer(buf)
		_, err := io.CopyN(buf, body, int64(body.Len()))
		if err != nil {
			mod.Logger().Warn().
				Int("type", int(typ)).
				Int("size", int(size)).
				String("name", proto.Nameof(m)).
				Error("error", err).
				Print("read message body error")
			return nil, err
		}
		switch s.ContentType() {
		case proto.ContentTypeProtobuf:
			err = buf.Unmarshal(m)
		case proto.ContentTypeText:
			err = json.Unmarshal(buf.Bytes(), m)
		default:
			err = proto.ErrUnsupportedContentType
		}
		if err != nil {
			mod.Logger().Warn().
				Int("type", int(typ)).
				String("name", proto.Nameof(m)).
				Error("error", err).
				Print("unmarshal typed message error")
			return nil, err
		}
	}
	return m, nil
}

func (mod *frontendModule) userKey(uid int64) string {
	return path.Join(mod.service.Config().Core.Project, frontend.UsersTable, strconv.FormatInt(uid, 10))
}

func (mod *frontendModule) setUserLogged(uid, sid int64, nx bool) (bool, error) {
	buf := make([]byte, 0, 32)
	buf = strconv.AppendInt(buf, mod.service.ID(), 10)
	buf = append(buf, ',')
	buf = strconv.AppendInt(buf, sid, 10)
	content := string(buf)
	ttl := time.Duration(mod.service.Config().UserTTL) * time.Second
	err := mod.service.Discovery().Register(context.Background(), "", mod.userKey(uid), content, nx, ttl)
	if err != nil {
		if discovery.IsExist(err) {
			return false, err
		}
		mod.Logger().Warn().
			Int64("uid", uid).
			Error("error", err).
			Print("register user error")
		return false, err
	}
	return true, nil
}

func (mod *frontendModule) delUserLogged(uid int64) (bool, error) {
	err := mod.service.Discovery().Unregister(context.Background(), "", mod.userKey(uid))
	if err != nil {
		mod.Logger().Warn().
			Int64("uid", uid).
			Error("error", err).
			Print("unregister user error")
		return false, err
	}
	return true, nil
}

// forward forwards message to other services
func (mod *frontendModule) forward(s *session, typ proto.Type, body proto.Body) error {
	content, err := ioutil.ReadAll(body)
	if err != nil {
		mod.Logger().Warn().
			Error("error", err).
			Int64("uid", s.getUid()).
			Int("type", int(typ)).
			Print("read body error")
		return err
	}
	return mod.service.Backend().Forward(s.getUid(), typ, content)
}

// ping handles Ping message
func (mod *frontendModule) ping(s *session, typ proto.Type, body proto.Body) error {
	if mod.service.Config().ForwardPing {
		return mod.forward(s, typ, body)
	} else if m, err := mod.unmarshal(s, typ, body); err != nil {
		return err
	} else {
		ttl := int64(mod.service.Config().UserTTL) * 1000
		if s.trySetLastUpdateSidTime(ttl/2, time.Now().UnixNano()/1e6) {
			_, err := mod.setUserLogged(s.getUid(), s.id, false)
			if err != nil {
				return err
			}
		}
		req := m.(*gatepb.Ping)
		mod.Logger().Debug().
			Int64("sid", s.id).
			String("content", req.Content).
			Print("received ping message")
		return s.send(&gatepb.Pong{
			Content: req.Content,
		})
	}
}

// login handles LoginReq message
func (mod *frontendModule) login(s *session, req *gatepb.LoginReq) error {
	cfg := mod.service.Config()
	if s.getState() == stateOverflow {
		s.send(&gatepb.LogoutRes{
			Reason: gatepb.KickoutReason_ReasonOverflow,
		})
		s.Close(nil)
		return nil
	}
	claims, err := mod.verifier.Verify(cfg.JWT.Issuer, req.Token)
	if err != nil {
		mod.Logger().Warn().
			Int64("sid", s.id).
			String("token", req.Token).
			Error("error", err).
			Print("verify token error")
		return err
	}
	mod.Logger().Debug().
		Int64("sid", s.id).
		Int64("uid", claims.Payload.ID).
		String("ip", claims.Payload.IP).
		Print("user logging")

	// overrides ip
	if claims.Payload.IP != "" {
		s.ip = claims.Payload.IP
	}

	if !mod.sessions.recordIP(s.id, s.ip) {
		mod.Logger().Warn().
			Int64("sid", s.id).
			Int64("uid", claims.Payload.ID).
			String("ip", claims.Payload.IP).
			Print("user login denied because of ip limited")
		return errors.New("ip limited")
	}

	s.setUser(user{
		token: claims.Payload,
	})

	if ok, err := mod.setUserLogged(claims.Payload.ID, s.id, true); err != nil {
		return err
	} else if !ok {
		mod.pendingSessions.Store(s.id, &pendingSession{
			uid: claims.Payload.ID,
		})
		s.setState(statePendingLogin)
		return mod.service.Backend().Login(claims.Payload, true)
	}
	return mod.afterLogin(s)
}

func (mod *frontendModule) retryLogin(sid int64, ps *pendingSession, now int64) (deleted bool) {
	s := mod.sessions.get(sid)
	if s == nil {
		mod.Logger().Debug().
			Int64("sid", sid).
			Int64("uid", ps.uid).
			Print("session not found when retry login")
		return true
	}
	if s.getState() != statePendingLogin {
		mod.Logger().Debug().
			Int64("sid", sid).
			Int64("uid", ps.uid).
			Int("state", int(statePendingLogin)).
			Print("session state is not pending")
		return true
	}
	if ok, err := mod.setUserLogged(ps.uid, sid, true); err != nil {
		mod.Logger().Debug().
			Int64("sid", sid).
			Int64("uid", ps.uid).
			Error("error", err).
			Print("set user logged failed")
		s.send(&gatepb.LogoutRes{
			Reason: gatepb.KickoutReason_ReasonLoginAnotherDevice,
		})
		s.Close(nil)
		return true
	} else if ok {
		mod.afterLogin(s)
		return true
	}
	if s.createdAt+int64(maxDurationForPendingSession/time.Millisecond) < now {
		mod.Logger().Debug().
			Int64("sid", sid).
			Int64("uid", ps.uid).
			Print("user login repeated")
		s.send(&gatepb.LogoutRes{
			Reason: gatepb.KickoutReason_ReasonLoginAnotherDevice,
		})
		s.Close(nil)
		return true
	}
	return false
}

func (mod *frontendModule) afterLogin(s *session) error {
	if !mod.sessions.mapping(s.getUid(), s.id) {
		mod.Logger().Warn().
			Int64("uid", s.getUid()).
			Int64("sid", s.id).
			Print("duplicated login")
		return errors.New("duplicated login")
	}
	s.setState(stateLogged)
	return mod.service.Backend().Login(s.getUser().token, false)
}

func (mod *frontendModule) logout(s *session, req *gatepb.LogoutReq) error {
	s.send(&gatepb.LogoutRes{
		Reason: gatepb.KickoutReason_ReasonUserLogout,
	})
	s.Close(nil)
	return nil
}

func (mod *frontendModule) afterLogout(s *session) error {
	uid := s.getUid()
	if uid <= 0 {
		return nil
	}
	return mod.service.Backend().Logout(uid)
}

// BroadcastAll implements module.Frontend BroadcastAll method
func (mod *frontendModule) BroadcastAll(content []byte) error {
	mod.sessions.broadcast(content, time.Now().UnixNano()/1e6)
	return nil
}

// Broadcast implements frontend.Module Broadcast method
func (mod *frontendModule) Broadcast(uids []int64, content []byte) error {
	for _, uid := range uids {
		s := mod.sessions.find(uid)
		if s != nil {
			s.Write(content)
		}
	}
	return nil
}

// Unicast implements frontend.Module Unicast method
func (mod *frontendModule) Unicast(uid int64, content []byte) error {
	s := mod.sessions.find(uid)
	if s == nil {
		mod.Logger().Debug().
			Int64("uid", uid).
			Print("send to user failed, session not found by uid")
		return nil
	}
	mod.Logger().Trace().
		Int64("uid", uid).
		Int64("sid", s.id).
		Int("size", len(content)).
		Print("send to user session")
	_, err := s.Write(content)
	return err
}

// Send implements frontend.Module Send method
func (mod *frontendModule) Send(uid int64, m proto.Message) error {
	s := mod.sessions.find(uid)
	if s == nil {
		mod.Logger().Debug().
			Int64("uid", uid).
			Print("send to user failed, session not found by uid")
		return nil
	}
	mod.Logger().Trace().
		Int64("uid", uid).
		Int64("sid", s.id).
		Int32("type", int32(m.Type())).
		String("name", proto.Nameof(m)).
		Print("send to user session")
	return s.send(m)
}

// Kickout implements frontend.Module Kickout method
func (mod *frontendModule) Kickout(uid int64, reason gatepb.KickoutReason) error {
	s := mod.sessions.find(uid)
	if s == nil {
		mod.Logger().Debug().
			Int64("uid", uid).
			Print("user session not found by uid")
		return nil
	}
	mod.Logger().Debug().
		Int64("uid", uid).
		Int64("sid", s.id).
		String("reason", reason.String()).
		Print("kickout user")
	s.send(&gatepb.Kickout{
		Reason: int32(reason),
	})
	s.Close(nil)
	return nil
}
