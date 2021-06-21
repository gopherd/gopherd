package internal

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/jwt"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/service/component"
	"github.com/gopherd/doge/service/discovery"

	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/module"
	"github.com/gopherd/gopherd/proto/gatepb"
)

type Service interface {
	ID() int64
	GetConfig() *config.Config
	Backend() module.Backend
	Discovery() discovery.Discovery
}

func NewComponent(service Service) *frontend {
	return newFrontend(service)
}

// frontend component
type frontend struct {
	*component.BaseComponent

	service  Service
	verifier *jwt.Verifier

	http struct {
		server   *http.Server
		listener net.Listener
	}

	maxConns      int
	maxConnsPerIP int
	nextSessionId int64

	mutex    sync.RWMutex
	uid2sid  map[int64]int64
	ips      map[string]int
	sessions map[int64]*session
	pendings map[int64]*pendingSession
}

func newFrontend(service Service) *frontend {
	return &frontend{
		BaseComponent: component.NewBaseComponent("frontend"),
		service:       service,
		uid2sid:       make(map[int64]int64),
		ips:           make(map[string]int),
		sessions:      make(map[int64]*session),
		pendings:      make(map[int64]*pendingSession),
	}
}

// Init overrides BaseComponent Init method
func (f *frontend) Init() error {
	if err := f.BaseComponent.Init(); err != nil {
		return err
	}
	cfg := f.service.GetConfig()
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
	addr := fmt.Sprintf("%s:%d", cfg.Net.Host, cfg.Net.Port)
	server, listener, err := httputil.ListenWebsocket(addr, "/", f.onOpen, time.Minute*3)
	if err != nil {
		return erron.Throw(err)
	}
	f.http.server = server
	f.http.listener = listener
	f.Logger().Info().String("addr", addr).Print("http server listening")

	return nil
}

// Start overrides BaseComponent Start method
func (f *frontend) Start() {
	f.BaseComponent.Start()
	go f.http.server.Serve(f.http.listener)
}

// Shutdown overrides BaseComponent Shutdown method
func (f *frontend) Shutdown() {
	f.BaseComponent.Shutdown()
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	for _, s := range f.sessions {
		if state := s.getState(); state == stateClosing || state == stateOverflow {
			continue
		}
		s.Close()
	}
}

func (f *frontend) size() int {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return len(f.sessions)
}

func (f *frontend) add(s *session) (n int, ok bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.sessions[s.id] = s
	n = len(f.sessions)
	if n < f.maxConns {
		ok = true
	} else {
		s.setState(stateOverflow)
	}
	return
}

func (f *frontend) remove(id int64) *session {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	s, ok := f.sessions[id]
	if !ok {
		return nil
	}
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

func (f *frontend) mapping(uid, sid int64) bool {
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

func (f *frontend) get(sid int64) *session {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.sessions[sid]
}

func (f *frontend) find(uid int64) *session {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	sid, ok := f.uid2sid[uid]
	if !ok {
		return nil
	}
	return f.sessions[sid]
}

func (f *frontend) recordIP(sid int64, ip string) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if n := f.ips[ip]; n < f.maxConnsPerIP {
		f.ips[ip] = n + 1
		return true
	}
	return false
}

func (f *frontend) clean(ttl, now int64) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	for sid, s := range f.sessions {
		if s.getLastKeepaliveTime()+ttl < now {
			f.Logger().Debug().Int64("sid", sid).Print("clean dead session")
			s.Close()
		}
	}
}

func (f *frontend) broadcast(data []byte, ttl, now int64) {
	f.mutex.RLock()
	defer f.mutex.Unlock()
	for sid, s := range f.sessions {
		if s.getLastKeepaliveTime()+ttl > now {
			if _, err := s.Write(data); err != nil {
				f.Logger().Warn().
					Int64("sid", sid).
					Error("error", err).
					Int("bytes", len(data)).
					Print("broadcast to session error")
			}
		}
	}
}

func (f *frontend) allocSessionId() int64 {
	return atomic.AddInt64(&f.nextSessionId, 1)
}

func (f *frontend) onOpen(ip string, conn net.Conn) {
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
func (f *frontend) onReady(sess *session) {
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
func (f *frontend) onClose(sess *session, err error) {
	f.Logger().Debug().Int64("sid", sess.id).Print("session closed")
	f.remove(sess.id)
}

// onMessage implements handler onMessage method
func (f *frontend) onMessage(sess *session, body proto.Body) error {
	n, typ, err := proto.PeekType(body)
	if err != nil {
		f.Logger().Debug().
			Int64("sid", sess.id).
			Int("bytes", body.Len()).
			Error("error", err).
			Print("session received an untyped message")
		return err
	} else {
		f.Logger().Trace().
			Int64("sid", sess.id).
			Int("bytes", body.Len()).
			Int("type", int(typ)).
			Print("session received a typed message")
	}
	var m proto.Message
	switch typ {
	case gatepb.PingType:
		if f.service.GetConfig().ForwardPing {
			err = f.forward(sess, typ, body)
		} else if m, err = f.unmarshal(n, typ, body); err == nil {
			err = f.ping(sess, m.(*gatepb.Ping))
		}
	case gatepb.LoginType:
		if m, err = f.unmarshal(n, typ, body); err == nil {
			err = f.login(sess, m.(*gatepb.Login))
		}
	case gatepb.LogoutType:
		if m, err = f.unmarshal(n, typ, body); err == nil {
			err = f.logout(sess, m.(*gatepb.Logout))
		}
	default:
		err = f.forward(sess, typ, body)
	}
	return err
}

func (f *frontend) unmarshal(discard int, typ proto.Type, body proto.Body) (proto.Message, error) {
	if _, err := body.Discard(discard); err != nil {
		return nil, err
	}
	m := proto.New(typ)
	if m == nil {
		return nil, proto.ErrUnrecognizedType
	}
	buf := proto.AllocBuffer()
	defer proto.FreeBuffer(buf)
	if _, err := io.CopyN(buf, body, int64(body.Len())); err != nil {
		return nil, err
	}
	if err := buf.Unmarshal(m); err != nil {
		f.Logger().Warn().
			Int("type", int(typ)).
			String("name", proto.Nameof(m)).
			Error("error", err).
			Print("unmarshal typed message error")
		return nil, err
	}
	return m, nil
}

func (f *frontend) setUserLogged(uid, sid int64) (bool, error) {
	var (
		name    = path.Join(f.service.GetConfig().Core.Project, module.UsersTable)
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

func (f *frontend) forward(sess *session, typ proto.Type, body proto.Body) error {
	return f.service.Backend().Forward(sess.getUid(), typ, body)
}

func (f *frontend) ping(sess *session, req *gatepb.Ping) error {
	pong := &gatepb.Pong{
		Content: req.Content,
	}
	return sess.send(pong)
}

func (f *frontend) login(sess *session, req *gatepb.Login) error {
	cfg := f.service.GetConfig()
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

func (f *frontend) logout(sess *session, req *gatepb.Logout) error {
	uid := sess.getUid()
	if uid <= 0 {
		return nil
	}
	return f.service.Backend().Logout(uid)
}

// BroadcastAll implements module.Frontend BroadcastAll method
func (f *frontend) BroadcastAll(content []byte) error {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	for _, sess := range f.sessions {
		sess.Write(content)
	}
	return nil
}

// Broadcast implements module.Frontend Broadcast method
func (f *frontend) Broadcast(uids []int64, content []byte) error {
	for _, uid := range uids {
		sess := f.find(uid)
		if sess != nil {
			sess.Write(content)
		}
	}
	return nil
}

// Send implements module.Frontend Send method
func (f *frontend) Send(uid int64, content []byte) error {
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
		Int("bytes", len(content)).
		Print("send to user session")
	sess.Write(content)
	return nil
}

// Kickout implements module.Frontend Kickout method
func (f *frontend) Kickout(uid int64, reason gatepb.KickoutReason) error {
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
	sess.Close()
	return nil
}
