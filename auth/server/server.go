package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/doge/service"
	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/config"
	"github.com/gopherd/gopherd/auth/handler"
	"github.com/gopherd/jwt"
	"github.com/gopherd/log"
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
	signer *jwt.Signer

	quit, wait chan struct{}
}

// New creates authd service
func New(cfg *config.Config) service.Service {
	prefix := cfg.Core.Name
	if prefix == "" {
		prefix = "authd"
	}
	s := &server{
		quit: make(chan struct{}),
		wait: make(chan struct{}),
	}
	s.BaseService = service.NewBaseService(s, cfg)
	s.internal.config = cfg
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
	cfg := s.internal.config
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

func (s *server) registerHTTPHandlers() {
	s.registeAPI("/auth/authorize", handler.Authorize)
	s.registeAPI("/auth/link", handler.Link)
	s.registeAPI("/auth/smscode", handler.SMSCode)
}

func (s *server) registeAPI(pattern string, h auth.Handler) {
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
	return s.BaseService.Busy()
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

func (s *server) Options() auth.Options {
	return s.Config().Options
}

func (s *server) Logger() *log.Logger {
	return log.GlobalLogger()
}

func (s *server) Provider(name string) (auth.Provider, error) {
	return nil, errors.New("TODO")
}

func (s *server) Response(w http.ResponseWriter, r *http.Request, data interface{}) error {
	return s.http.server.JSONResponse(w, r, data)
}

func (s *server) Signer() *jwt.Signer {
	return s.signer
}

func (s *server) GenerateSMSCode(channel int, ip, mobile string) (time.Duration, error) {
	return 0, errors.New("TODO")
}
