package internal

import (
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/service/component"
	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/module"
	"github.com/gopherd/gopherd/proto/gatepb"
	"github.com/mkideal/log"
)

type Service interface {
	GetConfig() *config.Config
	Frontend() module.Frontend
}

func NewComponent(service Service) *frontend {
	return newFrontend(service)
}

// frontend component
type frontend struct {
	*component.BaseComponent
	service Service

	maxConns      int
	maxConnsPerIP int

	nextSessionId int64
	mutex         sync.RWMutex
	sessions      map[int64]*session
	uid2sid       map[int64]int64
	ips           map[string]int
}

func newFrontend(service Service) *frontend {
	return &frontend{
		BaseComponent: component.NewBaseComponent("sessions"),
		service:       service,
		sessions:      make(map[int64]*session),
		uid2sid:       make(map[int64]int64),
		ips:           make(map[string]int),
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

	if cfg.Net.Port <= 0 {
		return erron.Throwf("invalid port: %d", cfg.Net.Port)
	}
	addr := fmt.Sprintf("%s:%d", cfg.Net.Host, cfg.Net.Port)
	if err := httputil.ListenAndServeWebsocket(addr, "/", f.onOpen, true); err != nil {
		return erron.Throw(err)
	}
	log.Info("listening on %s", addr)

	return nil
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
			log.Debug("clean dead session %d", sid)
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
				log.Warn("broadcast: write data to session %d error: %v", sid, err)
			}
		}
	}
}

func (f *frontend) allocSessionId() int64 {
	return atomic.AddInt64(&f.nextSessionId, 1)
}

func (f *frontend) onOpen(ip string, conn net.Conn) {
	id := f.allocSessionId()
	log.Debug("session %d connected from %s", id, ip)
	sess := newSession(id, ip, conn, f)
	// Blocked here
	sess.serve()
}

// onReady implements handler onReady method
func (f *frontend) onReady(sess *session) {
	n, ok := f.add(sess)
	if !ok {
		log.Warn("add session %d failed, current total %d sessions", sess.id, n)
	} else {
		log.Debug("session %d ready, current total %d sessions", sess.id, n)
	}
}

// onClose implements handler onClose method
func (f *frontend) onClose(sess *session, err error) {
	log.Debug("session %d closed", sess.id)
	f.remove(sess.id)
}

// onMessage implements handler onMessage method
func (f *frontend) onMessage(sess *session, body proto.Body) error {
	n, typ, err := proto.PeekType(body)
	if err != nil {
		log.Debug("session %d received an untyped message which has %d bytes: %v", sess.id, body.Len(), err)
		return err
	} else {
		log.Trace("session %d received a typed message %d which has %d bytes", sess.id, typ, body.Len())
	}
	var m proto.Message
	switch typ {
	case gatepb.PingType:
		if f.service.GetConfig().ForwardPing {
			err = f.forward(sess, typ, body)
		} else if m, err = f.unmarshal(n, typ, body); err == nil {
			err = f.onPing(sess, m.(*gatepb.Ping))
		}
	case gatepb.LoginType:
		if m, err = f.unmarshal(n, typ, body); err == nil {
			err = f.onLogin(sess, m.(*gatepb.Login))
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
		log.Warn("unmarshal typed message %d (%s) failed: %v", typ, proto.Nameof(m), err)
		return nil, err
	}
	return m, nil
}

func (f *frontend) forward(sess *session, typ proto.Type, body proto.Body) error {
	return nil
}

func (f *frontend) onPing(sess *session, req *gatepb.Ping) error {
	pong := &gatepb.Pong{
		Content: req.Content,
	}
	return sess.send(pong)
}

func (f *frontend) onLogin(sess *session, req *gatepb.Login) error {
	return nil
}

func (f *frontend) Broadcast(content []byte) error {
	return nil
}

func (f *frontend) BroadcastTo(uids []int64, content []byte) error {
	return nil
}
