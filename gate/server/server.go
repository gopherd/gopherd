package server

import (
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/proto"
	"github.com/gopherd/doge/service"

	"github.com/gopherd/gopherd/gate/backend"
	"github.com/gopherd/gopherd/gate/backend/backendmod"
	"github.com/gopherd/gopherd/gate/config"
	"github.com/gopherd/gopherd/gate/frontend"
	"github.com/gopherd/gopherd/gate/frontend/frontendmod"
)

type server struct {
	*service.BasicService

	internal struct {
		config *config.Config
	}

	quit, wait chan struct{}

	modules struct {
		frontend frontend.Module
		backend  backend.Module
	}
	arena *proto.Arena
}

// New creates gated service
func New(cfg *config.Config) service.Service {
	s := &server{
		quit: make(chan struct{}),
		wait: make(chan struct{}),
	}
	s.BasicService = service.NewBasicService(s, cfg)
	s.internal.config = cfg

	s.modules.frontend = s.AddModule(frontendmod.New(s)).(frontend.Module)
	s.modules.backend = s.AddModule(backendmod.New(s)).(backend.Module)

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

// Init overrides BasicService Init method
func (s *server) Init() error {
	if err := s.BasicService.Init(); err != nil {
		return erron.Throw(err)
	}
	return nil
}

// Start overrides BasicService Start method
func (s *server) Start() error {
	s.BasicService.Start()
	go s.run()
	return nil
}

// Shutdown overrides BasicService Shutdown method
func (s *server) Shutdown() error {
	close(s.quit)
	<-s.wait
	return s.BasicService.Shutdown()
}

func (s *server) Busy() bool {
	return s.BasicService.Busy() || s.modules.frontend.Busy() || s.modules.backend.Busy()
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
	s.BasicService.Update(now, dt)
}

func (s *server) Arena() *proto.Arena       { return s.arena }
func (s *server) Frontend() frontend.Module { return s.modules.frontend }
func (s *server) Backend() backend.Module   { return s.modules.backend }
