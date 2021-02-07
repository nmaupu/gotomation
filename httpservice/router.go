package httpservice

import (
	"fmt"
	"net/http"

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
}

type httpService struct {
	BindAddr string
	Port     int
	server   *http.Server
}

// HTTPServer returns the HTTP server singleton
func HTTPServer() HTTPService {
	return &httpServer
}

// InitHTTPServer inits HTTP server singleton
func InitHTTPServer(bindAddr string, port int) {
	httpServer = httpService{
		BindAddr: bindAddr,
		Port:     port,
	}
}

// Start starts the HTTP service
func (s *httpService) Start() error {
	l := logging.NewLogger("HTTPService.Start")
	if s.Port == 0 {
		l.Warn().Msgf("No port defined for HTTP server, using %d", DefaultHTTPPort)
		s.Port = DefaultHTTPPort
	}

	l.Info().Msg("Starting HTTP server")
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", controllers.HealthHandler)
	router.GET("/google-validate", controllers.GoogleWebTokenHandler)
	router.GET("/coords", controllers.CoordsHandler)
	router.GET("/sun", controllers.SunriseSunsetHandler)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.BindAddr, s.Port),
		Handler: router,
	}

	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		s.server.ListenAndServe()
	}()

	return nil
}

// Stop stops the HTTP service and free resources
func (s *httpService) Stop() {
	l := logging.NewLogger("HTTPService.Stop")
	if s.server == nil {
		l.Warn().Msg("HTTP server is not initialized")
		return
	}

	l.Trace().Msg("Stopping HTTP server")
	s.server.Shutdown(context.TODO())

}

// GetName returns the name of this runnable object
func (s *httpService) GetName() string {
	return "HTTPService"
}
