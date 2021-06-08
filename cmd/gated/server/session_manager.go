package server

import (
	"sync"

	"github.com/gopherd/doge/component"
	"github.com/mkideal/log"
)

type pendingSession struct {
	uid  int64
	meta uint32
}

type sessionManager struct {
	*component.BaseComponent

	maxConns      int
	maxConnsPerIP int

	context interface {
		GetConfig() *Config
	}

	mutex    sync.RWMutex
	sessions map[int64]*session
	uid2sid  map[int64]int64
	ips      map[string]int
}

func newSessionManager(server *server) *sessionManager {
	return &sessionManager{
		BaseComponent: component.NewBaseComponent("session_manager"),
		sessions:      make(map[int64]*session),
		uid2sid:       make(map[int64]int64),
		ips:           make(map[string]int),
		context:       server,
	}
}

// Init overrides BaseComponent Init method
func (m *sessionManager) Init() error {
	if err := m.BaseComponent.Init(); err != nil {
		return err
	}
	cfg := m.context.GetConfig()
	m.maxConns = cfg.MaxConns
	m.maxConnsPerIP = cfg.MaxConnsPerIP
	return nil
}

// Shutdown overrides BaseComponent Shutdown method
func (m *sessionManager) Shutdown() {
	m.BaseComponent.Shutdown()
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, s := range m.sessions {
		if state := s.getState(); state == stateClosing || state == stateOverflow {
			continue
		}
		s.Close()
	}
}

func (m *sessionManager) size() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.sessions)
}

func (m *sessionManager) add(s *session) (n int, ok bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.sessions[s.id] = s
	n = len(m.sessions)
	if n < m.maxConns {
		ok = true
	} else {
		s.setState(stateOverflow)
	}
	return
}

func (m *sessionManager) remove(id int64) *session {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil
	}
	ip := s.ip
	if n, ok := m.ips[ip]; n > 1 {
		m.ips[ip] = n - 1
	} else if ok {
		delete(m.ips, ip)
	}
	if uid := s.getUid(); uid > 0 {
		delete(m.uid2sid, uid)
	}
	return s
}

func (m *sessionManager) mapping(uid, sid int64) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if old, ok := m.uid2sid[uid]; ok {
		if sid != old {
			ok = false
		}
		return ok
	}
	m.uid2sid[uid] = sid
	return true
}

func (m *sessionManager) get(sid int64) *session {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.sessions[sid]
}

func (m *sessionManager) find(uid int64) *session {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	sid, ok := m.uid2sid[uid]
	if !ok {
		return nil
	}
	return m.sessions[sid]
}

func (m *sessionManager) recordIP(sid int64, ip string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if n := m.ips[ip]; n < m.maxConnsPerIP {
		m.ips[ip] = n + 1
		return true
	}
	return false
}

func (m *sessionManager) clean(ttl, now int64) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for sid, s := range m.sessions {
		if s.getLastKeepaliveTime()+ttl < now {
			log.Debug("clean dead session %d", sid)
			s.Close()
		}
	}
}

func (m *sessionManager) broadcast(data []byte, ttl, now int64) {
	m.mutex.RLock()
	defer m.mutex.Unlock()
	for sid, s := range m.sessions {
		if s.getLastKeepaliveTime()+ttl > now {
			if _, err := s.Write(data); err != nil {
				log.Warn("broadcast: write data to session %d error: %v", sid, err)
			}
		}
	}
}
