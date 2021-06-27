package server

import (
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/service"

	"github.com/gopherd/gopherd/cmd/gated/backend"
	"github.com/gopherd/gopherd/cmd/gated/backend/backendinternal"
	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/frontend"
	"github.com/gopherd/gopherd/cmd/gated/frontend/frontendinternal"
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
		config *config.Config
	}

	quit, wait chan struct{}

	components struct {
		frontend frontend.Frontend
		backend  backend.Backend
	}
}

// New creates gated service
func New(cfg *config.Config) service.Service {
	s := &server{
		quit: make(chan struct{}),
		wait: make(chan struct{}),
	}
	s.BaseService = service.NewBaseService(s, cfg)
	s.internal.config = cfg

	s.components.frontend = s.AddComponent(frontendinternal.New(s)).(frontend.Frontend)
	s.components.backend = s.AddComponent(backendinternal.New(s)).(backend.Backend)

	return s
}

// Config atomically loads the config
func (s *server) Config() *config.Config {
	return (*config.Config)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.internal.config))))
}

// RewriteConfig implements Service RewriteConfig method to atomically stores the config
func (s *server) RewriteConfig(cfg unsafe.Pointer) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.internal.config)), cfg)
}

// Init overrides BaseService Init method
func (s *server) Init() error {
	if err := s.BaseService.Init(); err != nil {
		return erron.Throw(err)
	}
	return nil
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

func (s *server) Frontend() frontend.Frontend { return s.components.frontend }
func (s *server) Backend() backend.Backend    { return s.components.backend }
