package httpservice

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/httpservice/controllers"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/routines"
	"golang.org/x/net/context"
)

var (
	httpServer httpService
)

const (
	// DefaultHTTPPort is the default port used to listen to incoming HTTP requests
	DefaultHTTPPort = 6265
)

// HTTPService is Gotomation's HTTP server
type HTTPService interface {
	routines.Runnable
	AddExtraHandlers(getHandlers ...GinConfigHandlers)
}

type httpService struct {
	BindAddr string
	Port     int
	server   *http.Server

	started        bool
	mutexStopStart sync.Mutex

	router *gin.Engine
}

// HTTPServer returns the HTTP server singleton
func HTTPServer() HTTPService {
	return &httpServer
}

// InitHTTPServer inits HTTP server singleton
func InitHTTPServer(bindAddr string, port int, getExtraHandlers ...GinConfigHandlers) error {
	httpServer = httpService{
		BindAddr: bindAddr,
		Port:     port,
		router:   gin.New(),
	}

	httpServer.router.Use(gin.Recovery())
	httpServer.router.GET("/health", controllers.HealthHandler)
	httpServer.router.GET("/google-validate", controllers.GoogleWebTokenHandler)
	httpServer.router.GET("/coords", controllers.CoordsHandler)
	httpServer.router.GET("/sun", controllers.SunriseSunsetHandler)
	httpServer.AddExtraHandlers(getExtraHandlers...)
	return nil
}

// GinConfigHandlers stores gin handlers configuration
type GinConfigHandlers struct {
	Path     string
	Handlers []gin.HandlerFunc
}

func (s *httpService) AddExtraHandlers(getHandlers ...GinConfigHandlers) {
	if s.router == nil {
		return
	}

	// Configuring extra handlers
	for _, eh := range getHandlers {
		s.router.GET(eh.Path, eh.Handlers...)
	}
}

func (s *httpService) IsAutoStart() bool {
	return true
}

// Start starts the HTTP service
func (s *httpService) Start() error {
	s.mutexStopStart.Lock()
	defer s.mutexStopStart.Unlock()
	if s.started {
		return nil
	}

	l := logging.NewLogger("HTTPService.Start")
	if s.Port == 0 {
		l.Warn().Msgf("No port defined for HTTP server, using %d", DefaultHTTPPort)
		s.Port = DefaultHTTPPort
	}

	l.Info().Msg("Starting HTTP server")
	gin.SetMode(gin.ReleaseMode)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.BindAddr, s.Port),
		Handler: s.router,
	}

	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		s.server.ListenAndServe()
	}()

	s.started = true
	return nil
}

// Stop stops the HTTP service and free resources
func (s *httpService) Stop() {
	s.mutexStopStart.Lock()
	defer s.mutexStopStart.Unlock()

	if !s.started {
		return
	}

	l := logging.NewLogger("HTTPService.Stop")
	if s.server == nil {
		l.Warn().Msg("HTTP server is not initialized")
		return
	}

	l.Trace().Msg("Stopping HTTP server")
	s.server.Shutdown(context.Background())
	s.started = false
}

func (s *httpService) IsStarted() bool {
	s.mutexStopStart.Lock()
	defer s.mutexStopStart.Unlock()
	return s.started
}

// GetName returns the name of this runnable object
func (s *httpService) GetName() string {
	return "HTTPService"
}
