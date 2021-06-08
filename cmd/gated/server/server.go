package server

import (
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gopherd/doge/service"
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
		config *Config
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
	return s.BaseService.Init()
}

// Start overrides BaseService Start method
func (s *server) Start() error {
	s.BaseService.Start()
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
