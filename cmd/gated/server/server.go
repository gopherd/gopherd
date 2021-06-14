package server

import (
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/jwt"
	"github.com/gopherd/doge/service"

	"github.com/gopherd/gopherd/cmd/gated/config"
	"github.com/gopherd/gopherd/cmd/gated/module"
	"github.com/gopherd/gopherd/cmd/gated/module/backend"
	"github.com/gopherd/gopherd/cmd/gated/module/frontend"
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
		frontend module.Frontend
		backend  module.Backend
	}
}

// New creates gated service
func New() service.Service {
	cfg := config.New()
	s := &server{
		BaseService: service.NewBaseService(cfg),
		quit:        make(chan struct{}),
		wait:        make(chan struct{}),
	}

	s.internal.config = cfg

	s.components.frontend = s.AddComponent(frontend.NewComponent(s)).(module.Frontend)
	s.components.backend = s.AddComponent(backend.NewComponent(s)).(module.Backend)

	return s
}

// GetConfig atomically gets the config
func (s *server) GetConfig() *config.Config {
	return (*config.Config)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.internal.config))))
}

// SetConfig atomically updates the config
func (s *server) SetConfig(cfg unsafe.Pointer) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.internal.config)), cfg)
}

// Init overrides BaseService Init method
func (s *server) Init() error {
	if err := s.BaseService.Init(); err != nil {
		return erron.Throw(err)
	}
	if err := jwt.LoadKeyFile(s.GetConfig().JWT.Key); err != nil {
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

func (s *server) Frontend() module.Frontend { return s.components.frontend }
func (s *server) Backend() module.Backend   { return s.components.backend }
