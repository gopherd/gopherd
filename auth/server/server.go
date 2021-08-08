package server

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/doge/service"
	"github.com/gopherd/jwt"
	"github.com/gopherd/log"

	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/account"
	"github.com/gopherd/gopherd/auth/config"
	"github.com/gopherd/gopherd/auth/geo"
	"github.com/gopherd/gopherd/auth/handler"
	"github.com/gopherd/gopherd/auth/oos"
	"github.com/gopherd/gopherd/auth/provider"
	"github.com/gopherd/gopherd/auth/sms"
)

type server struct {
	*service.BaseService

	internal struct {
		config *config.Config
	}
	http struct {
		listener net.Listener
		server   *httputil.HTTPServer
	}
	signer  *jwt.Signer
	modules struct {
		oos     auth.OOSModule
		account auth.AccountModule
		sms     auth.SMSModule
		geo     auth.GeoModule
	}

	providersMu sync.RWMutex
	providers   map[string]provider.Provider

	quit, wait chan struct{}
}

// New creates authd service
func New(cfg *config.Config) service.Service {
	prefix := cfg.Core.Name
	if prefix == "" {
		prefix = "authd"
	}
	s := &server{
		providers: make(map[string]provider.Provider),
		quit:      make(chan struct{}),
		wait:      make(chan struct{}),
	}
	s.BaseService = service.NewBaseService(s, cfg)
	s.internal.config = cfg
	s.modules.oos = s.AddModule(oos.New(s)).(auth.OOSModule)
	s.modules.account = s.AddModule(account.New(s)).(auth.AccountModule)
	s.modules.sms = s.AddModule(sms.New(s)).(auth.SMSModule)
	s.modules.geo = s.AddModule(geo.New(s)).(auth.GeoModule)
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
	err := s.BaseService.Init()
	if err != nil {
		return erron.Throw(err)
	}
	cfg := s.Config()
	s.signer, err = jwt.NewSigner(cfg.JWT.Filename, cfg.JWT.KeyId)
	if err != nil {
		return erron.Throwf("new signer from file %q error %w", cfg.JWT.Filename, err)
	}

	s.http.server = httputil.NewHTTPServer(cfg.HTTP)
	s.http.listener, err = s.http.server.Listen()
	if err != nil {
		return erron.Throwf("listen %s error %w", s.http.server.Addr(), err)
	}
	return nil
}

// Start overrides BaseService Start method
func (s *server) Start() error {
	s.BaseService.Start()
	s.registerHTTPHandlers()
	go s.http.server.Serve(s.http.listener)
	go s.run()
	return nil
}

// Shutdown overrides BaseService Shutdown method
func (s *server) Shutdown() error {
	s.shutdownHTTPServer()
	close(s.quit)
	<-s.wait
	return s.BaseService.Shutdown()
}

func or(x, y string) string {
	if x != "" {
		return x
	}
	return y
}

func (s *server) registerHTTPHandlers() {
	routers := s.Config().Options.Routers
	s.handleFunc(or(routers.Authorize, "/auth/authorize"), handler.Authorize)
	s.handleFunc(or(routers.Link, "/auth/link"), handler.Link)
	s.handleFunc(or(routers.SMSCode, "/auth/smscode"), handler.SMSCode)
}

func (s *server) handleFunc(pattern string, h func(auth.Service, http.ResponseWriter, *http.Request)) {
	s.http.server.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		h(s, w, r)
	})
}

func (s *server) shutdownHTTPServer() {
	if s.http.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		s.http.server.Shutdown(ctx)
	}
}

func (s *server) Busy() bool {
	return s.BaseService.Busy() || (s.http.server != nil && s.http.server.NumHandling() > 0)
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

func (s *server) Options() *auth.Options {
	return &s.Config().Options
}

func (s *server) Logger() *log.Logger {
	return log.GlobalLogger()
}

func (s *server) Provider(name string) (provider.Provider, error) {
	if p, ok := s.getProvider(name); ok {
		return p, nil
	}
	p, err := s.createProvider(name)
	if err != nil {
		return nil, err
	}
	s.providersMu.Lock()
	if old, ok := s.providers[name]; ok {
		s.providersMu.Unlock()
		p.Close()
		return old, nil
	}
	defer s.providersMu.Unlock()
	s.providers[name] = p
	return p, nil
}

func (s *server) getProvider(name string) (provider.Provider, bool) {
	s.providersMu.RLock()
	defer s.providersMu.RUnlock()
	p, ok := s.providers[name]
	return p, ok
}

func (s *server) createProvider(name string) (provider.Provider, error) {
	cfg := s.Config()
	if cfg.Proviers == nil {
		return nil, provider.ErrProviderNotFound
	}
	var source string
	if s, ok := cfg.Proviers[name]; !ok {
		return nil, provider.ErrProviderNotFound
	} else {
		source = s
	}
	return provider.Open(name, source)
}

func (s *server) Signer() *jwt.Signer {
	return s.signer
}

func (s *server) OOSModule() auth.OOSModule         { return s.modules.oos }
func (s *server) AccountModule() auth.AccountModule { return s.modules.account }
func (s *server) SMSModule() auth.SMSModule         { return s.modules.sms }
func (s *server) GeoModule() auth.GeoModule         { return s.modules.geo }
