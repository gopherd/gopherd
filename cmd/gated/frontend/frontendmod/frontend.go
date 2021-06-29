package frontendmod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/gopherd/jwt"

	"github.com/gopherd/gopherd/cmd/gated/backend"
	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/frontend"
	"github.com/gopherd/gopherd/proto/gatepb"
)

// New returns a frontend moudle
func New(service Service) module.Module {
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
	service Service

	verifier *jwt.Verifier
	server   interface{ Serve(net.Listener) error }
	listener net.Listener

	maxConns      int
	maxConnsPerIP int
	nextSessionId int64

	mutex    sync.RWMutex
	uid2sid  map[int64]int64
	ips      map[string]int
	sessions map[int64]*session
	pendings map[int64]*pendingSession
}

func newFrontendModule(service Service) *frontendModule {
	return &frontendModule{
		BaseModule: module.NewBaseModule("frontend"),
		service:    service,
		uid2sid:    make(map[int64]int64),
		ips:        make(map[string]int),
		sessions:   make(map[int64]*session),
		pendings:   make(map[int64]*pendingSession),
	}
}

// Init overrides BaseModule Init method
func (f *frontendModule) Init() error {
	if err := f.BaseModule.Init(); err != nil {
		return err
	}
	cfg := f.service.Config()
	f.maxConns = cfg.MaxConns
	f.maxConnsPerIP = cfg.MaxConnsPerIP

	if verifier, err := jwt.NewVerifier(cfg.JWT.Filename, cfg.JWT.KeyId); err != nil {
		return erron.Throw(err)
	} else {
		f.verifier = verifier
	}

	if cfg.Net.Port <= 0 {
		return erron.Throwf("invalid port: %d", cfg.Net.Port)
	}
	addr := fmt.Sprintf("%s:%d", cfg.Net.Bind, cfg.Net.Port)
	keepalive := time.Duration(cfg.Net.Keepalive) * time.Second
	if cfg.Net.Protocol == "tcp" {
		server, listener, err := netutil.ListenTCP(addr, f.onOpen, keepalive)
		if err != nil {
			return erron.Throw(err)
		}
		f.server = server
		f.listener = listener
	} else {
		server, listener, err := httputil.ListenWebsocket(addr, "/", f.onOpen, keepalive)
		if err != nil {
			return erron.Throw(err)
		}
		f.server = server
		f.listener = listener
	}
	f.Logger().Info().
		String("protocol", cfg.Net.Protocol).
		String("addr", addr).
		Print("listening")

	return nil
}

// Start overrides BaseModule Start method
func (f *frontendModule) Start() {
	f.BaseModule.Start()
	go f.server.Serve(f.listener)
}

// Shutdown overrides BaseModule Shutdown method
func (f *frontendModule) Shutdown() {
	f.BaseModule.Shutdown()
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	for _, s := range f.sessions {
		if state := s.getState(); state == stateClosing || state == stateOverflow {
			continue
		}
		s.Close(nil)
	}
}

func (f *frontendModule) size() int {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return len(f.sessions)
}

func (f *frontendModule) add(s *session) (n int, ok bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.sessions[s.id] = s
	n = len(f.sessions)
	if f.maxConns == 0 || n < f.maxConns {
		ok = true
	} else {
		s.setState(stateOverflow)
	}
	return
}

func (f *frontendModule) remove(id int64) *session {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	s, ok := f.sessions[id]
	if !ok {
		return nil
	}
	delete(f.sessions, id)

	ip := s.ip
	if n, ok := f.ips[ip]; n > 1 {
		f.ips[ip] = n - 1
	} else if ok {
		delete(f.ips, ip)
	}
	if uid := s.getUid(); uid > 0 {
		delete(f.uid2sid, uid)
	}
	return s
}

func (f *frontendModule) mapping(uid, sid int64) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if old, ok := f.uid2sid[uid]; ok {
		if sid != old {
			ok = false
		}
		return ok
	}
	f.uid2sid[uid] = sid
	return true
}

func (f *frontendModule) get(sid int64) *session {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.sessions[sid]
}

func (f *frontendModule) find(uid int64) *session {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	sid, ok := f.uid2sid[uid]
	if !ok {
		return nil
	}
	return f.sessions[sid]
}

func (f *frontendModule) recordIP(sid int64, ip string) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if n := f.ips[ip]; n < f.maxConnsPerIP {
		f.ips[ip] = n + 1
		return true
	}
	return false
}

func (f *frontendModule) clean(ttl, now int64) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	for sid, s := range f.sessions {
		if s.getLastKeepaliveTime()+ttl < now {
			f.Logger().Debug().Int64("sid", sid).Print("clean dead session")
			s.Close(nil)
		}
	}
}

func (f *frontendModule) broadcast(data []byte, ttl, now int64) {
	f.mutex.RLock()
	defer f.mutex.Unlock()
	for sid, s := range f.sessions {
		if s.getLastKeepaliveTime()+ttl > now {
			if _, err := s.Write(data); err != nil {
				f.Logger().Warn().
					Int64("sid", sid).
					Error("error", err).
					Int("size", len(data)).
					Print("broadcast to session error")
			}
		}
	}
}

func (f *frontendModule) allocSessionId() int64 {
	return atomic.AddInt64(&f.nextSessionId, 1)
}

func (f *frontendModule) onOpen(ip string, conn net.Conn) {
	sid := f.allocSessionId()
	f.Logger().Debug().
		Int64("sid", sid).
		String("ip", ip).
		Print("session connected")
	sess := newSession(f.Logger(), sid, ip, conn, f)
	// Blocked here
	sess.serve()
}

// onReady implements handler onReady method
func (f *frontendModule) onReady(sess *session) {
	n, ok := f.add(sess)
	if !ok {
		f.Logger().Warn().
			Int64("sid", sess.id).
			Int("sessions", n).
			Print("add session failed")
	} else {
		f.Logger().Debug().
			Int64("sid", sess.id).
			Int("sessions", n).
			Print("session ready")
	}
}

// onClose implements handler onClose method
func (f *frontendModule) onClose(sess *session, err error) {
	f.Logger().Debug().Int64("sid", sess.id).Print("session closed")
	f.remove(sess.id)
}

// onMessage implements handler onMessage method
func (f *frontendModule) onMessage(sess *session, typ proto.Type, body proto.Body) error {
	f.Logger().Trace().
		Int64("sid", sess.id).
		Int("size", body.Len()).
		Int("type", int(typ)).
		Print("session received a typed message")
	var (
		m   proto.Message
		err error
	)
	switch typ {
	case gatepb.PingType:
		if f.service.Config().ForwardPing {
			err = f.forward(sess, typ, body)
		} else if m, err = f.unmarshal(sess, typ, body); err == nil {
			err = f.ping(sess, m.(*gatepb.Ping))
		}
	case gatepb.LoginType:
		if m, err = f.unmarshal(sess, typ, body); err == nil {
			err = f.login(sess, m.(*gatepb.Login))
		}
	case gatepb.LogoutType:
		if m, err = f.unmarshal(sess, typ, body); err == nil {
			err = f.logout(sess, m.(*gatepb.Logout))
		}
	default:
		err = f.forward(sess, typ, body)
	}
	if err != nil {
		f.Logger().Warn().
			Int64("sid", sess.id).
			Int("type", int(typ)).
			Error("error", err).
			Print("session handle message error")
	}
	return err
}

// onCommand implements handler onCommand method
func (f *frontendModule) onCommand(sess *session, cmd *resp.Command) error {
	c := commands[strings.ToLower(cmd.Name())]
	if c == nil {
		typ, err := proto.ParseType(cmd.Name())
		if err != nil {
			if errors.Is(err, proto.ErrTypeOverflow) {
				return err
			}
			return errorln(sess, "command "+cmd.Name()+" not found, run command to list all supported commands")
		}
		switch cmd.NumArg() {
		case 0:
			return f.onMessage(sess, typ, proto.Text(nil))
		case 1:
			return f.onMessage(sess, typ, proto.Text(cmd.Arg(0).Value()))
		default:
			return resp.ErrNumberOfArguments
		}
	}
	return c.run(f, sess, cmd)
}

func (f *frontendModule) unmarshal(sess *session, typ proto.Type, body proto.Body) (proto.Message, error) {
	m := proto.New(typ)
	if m == nil {
		return nil, proto.ErrUnrecognizedType
	}
	if size := body.Len(); size > 0 {
		buf := proto.AllocBuffer()
		defer proto.FreeBuffer(buf)
		_, err := io.CopyN(buf, body, int64(body.Len()))
		if err != nil {
			f.Logger().Warn().
				Int("type", int(typ)).
				Int("size", int(size)).
				String("name", proto.Nameof(m)).
				Error("error", err).
				Print("read message body error")
			return nil, err
		}
		switch sess.ContentType() {
		case proto.ContentTypeProtobuf:
			err = buf.Unmarshal(m)
		case proto.ContentTypeText:
			err = json.Unmarshal(buf.Bytes(), m)
		default:
			err = proto.ErrUnsupportedContentType
		}
		if err != nil {
			f.Logger().Warn().
				Int("type", int(typ)).
				String("name", proto.Nameof(m)).
				Error("error", err).
				Print("unmarshal typed message error")
			return nil, err
		}
	}
	return m, nil
}

func (f *frontendModule) setUserLogged(uid, sid int64) (bool, error) {
	var (
		name    = path.Join(f.service.Config().Core.Project, frontend.UsersTable)
		content = make([]byte, 0, 32)
	)
	content = strconv.AppendInt(content, f.service.ID(), 10)
	content = append(content, ',')
	content = strconv.AppendInt(content, sid, 10)
	err := f.service.Discovery().Register(context.TODO(), name, strconv.FormatInt(uid, 10), string(content), true)
	if err != nil {
		if discovery.IsExist(err) {
			return false, nil
		}
		f.Logger().Warn().
			Int64("uid", uid).
			Error("error", err).
			Print("register user error")
		return false, err
	}
	return true, nil
}

func (f *frontendModule) forward(sess *session, typ proto.Type, body proto.Body) error {
	return f.service.Backend().Forward(sess.getUid(), typ, body)
}

func (f *frontendModule) ping(sess *session, req *gatepb.Ping) error {
	f.Logger().Debug().
		Int64("sid", sess.id).
		String("content", req.Content).
		Print("received ping message")
	pong := &gatepb.Pong{
		Content: req.Content,
	}
	return sess.send(pong)
}

func (f *frontendModule) login(sess *session, req *gatepb.Login) error {
	cfg := f.service.Config()
	claims, err := f.verifier.Verify(cfg.JWT.Issuer, req.Token)
	if err != nil {
		return err
	}
	uid, sid := claims.Uid, sess.id
	if ok, err := f.setUserLogged(uid, sid); err != nil {
		return err
	} else if !ok {
		// TODO: add to pendings
		return nil
	}
	return f.service.Backend().Login(uid, claims, req.Userdata)
}

func (f *frontendModule) logout(sess *session, req *gatepb.Logout) error {
	uid := sess.getUid()
	if uid <= 0 {
		return nil
	}
	return f.service.Backend().Logout(uid)
}

// BroadcastAll implements module.Frontend BroadcastAll method
func (f *frontendModule) BroadcastAll(content []byte) error {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	for _, sess := range f.sessions {
		sess.Write(content)
	}
	return nil
}

// Broadcast implements module.Frontend Broadcast method
func (f *frontendModule) Broadcast(uids []int64, content []byte) error {
	for _, uid := range uids {
		sess := f.find(uid)
		if sess != nil {
			sess.Write(content)
		}
	}
	return nil
}

// Send implements module.Frontend Send method
func (f *frontendModule) Send(uid int64, content []byte) error {
	sess := f.find(uid)
	if sess == nil {
		f.Logger().Debug().
			Int64("uid", uid).
			Print("send to user failed, session not found by uid")
		return nil
	}
	f.Logger().Trace().
		Int64("uid", uid).
		Int64("sid", sess.id).
		Int("size", len(content)).
		Print("send to user session")
	sess.Write(content)
	return nil
}

// Kickout implements module.Frontend Kickout method
func (f *frontendModule) Kickout(uid int64, reason gatepb.KickoutReason) error {
	sess := f.find(uid)
	if sess == nil {
		f.Logger().Debug().
			Int64("uid", uid).
			Print("user session not found by uid")
		return nil
	}
	f.Logger().Debug().
		Int64("uid", uid).
		Int64("sid", sess.id).
		Any("reason", reason).
		Print("kickout user")
	kickout := &gatepb.Kickout{
		Reason: int32(reason),
	}
	sess.send(kickout)
	sess.Close(nil)
	return nil
}
