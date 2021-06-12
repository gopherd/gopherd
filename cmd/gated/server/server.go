package server

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gopherd/doge/jwt"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/doge/net/netutil"
	"github.com/gopherd/doge/service"
	"github.com/mkideal/log"

	_ "github.com/gopherd/gopherd/proto/gatepb"
)

const (
	kMaxDurationForPendingSession = time.Second * 5
	kHandlePendingSessionInterval = time.Millisecond * 200
	kCleanDeadSessionInterval     = time.Minute
	kUserInfoTTLRatio             = 750 // 750/1000
)

type server struct {
	*service.BaseService

	internal struct {
		config        *Config
		nextSessionId int64
	}

	quit, wait chan struct{}

	// components list all components of ram
	components struct {
		sessionManager *sessionManager
		// more...
	}
}

// New creates gated service
func New() service.Service {
	cfg := NewConfig()
	s := &server{
		BaseService: service.NewBaseService(cfg),
		quit:        make(chan struct{}),
		wait:        make(chan struct{}),
	}

	s.internal.config = cfg

	s.components.sessionManager = s.AddComponent(
		newSessionManager(s),
	).(*sessionManager)

	return s
}

// GetConfig atomically gets the config
func (s *server) GetConfig() *Config {
	return (*Config)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.internal.config))))
}

// SetConfig atomically updates the config
func (s *server) SetConfig(cfg unsafe.Pointer) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.internal.config)), cfg)
}

// Init overrides BaseService Init method
func (s *server) Init() error {
	if err := s.BaseService.Init(); err != nil {
		return err
	}
	if err := jwt.LoadKeyFile(s.GetConfig().JWT.Key); err != nil {
		return err
	}
	return nil
}

// Start overrides BaseService Start method
func (s *server) Start() error {
	s.BaseService.Start()

	cfg := s.GetConfig()
	if cfg.Net.Port <= 0 {
		return fmt.Errorf("invalid port: %d", cfg.Net.Port)
	}
	addr := fmt.Sprintf("%s:%d", cfg.Net.Host, cfg.Net.Port)
	if err := httputil.ListenAndServeWebsocket(addr, "/", s.onOpen, true); err != nil {
		return err
	}
	log.Info("listening on %s", addr)

	go s.run()
	return nil
}

// Shutdown overrides BaseService Shutdown method
func (s *server) Shutdown() error {
	close(s.quit)
	<-s.wait
	s.BaseService.Shutdown()
	return nil
}

func (s *server) allocSessionId() int64 {
	return atomic.AddInt64(&s.internal.nextSessionId, 1)
}

func (s *server) onOpen(ip string, conn net.Conn) {
	id := s.allocSessionId()
	log.Debug("session %d connected from %s", id, ip)
	sess := newSession(id, ip, conn, s)
	// Blocked here
	sess.serve()
}

// onReady implements handler onReady method
func (s *server) onReady(sess *session) {
	n, ok := s.components.sessionManager.add(sess)
	if !ok {
		log.Warn("add session %d failed, current total %d sessions", sess.id, n)
	} else {
		log.Debug("session %d ready, current total %d sessions", sess.id, n)
	}
}

// onClose implements handler onClose method
func (s *server) onClose(sess *session, err error) {
	log.Debug("session %d closed", sess.id)
	s.components.sessionManager.remove(sess.id)
}

// onMessage implements handler onMessage method
func (s *server) onMessage(sess *session, body netutil.Body) error {
	log.Trace("session %d received a message which has %d bytes", sess, body.Len())
	// (TODO): handle the message body
	return nil
}

// run runs service's main loop
func (s *server) run() {
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	lastUpdatedAt := time.Now()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			s.onUpdate(now, now.Sub(lastUpdatedAt))
			lastUpdatedAt = now
		case <-s.quit:
			close(s.wait)
			return
		}
	}
}

func (s *server) onUpdate(now time.Time, dt time.Duration) {
	s.BaseService.Update(now, dt)
}
