package httpservice

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/go-homedir"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/httpservice/controllers"
	"github.com/nmaupu/gotomation/logging"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const (
	// DefaultHTTPPort is the default port used to listen to incoming HTTP requests
	DefaultHTTPPort = 6265
)

// HTTPService is Gotomation's HTTP server
type HTTPService struct {
	GoogleConfig *controllers.GoogleConfig
	BindAddr     string
	Port         int
	server       *http.Server
}

// NewHTTPService returns a pointer to a new HTTPService object
func NewHTTPService(config *oauth2.Config) (*HTTPService, error) {
	errMsg := "Unable to get home directory to store Google's token"
	hdir, err := homedir.Dir()
	if err != nil {
		return nil, errors.Wrap(err, errMsg)
	}
	expHomedir, err := homedir.Expand(hdir)
	if err != nil {
		return nil, errors.Wrap(err, errMsg)
	}
	tokenFilePath := fmt.Sprintf("%s/.gotomation-google-token.json", expHomedir)

	return &HTTPService{
		GoogleConfig: &controllers.GoogleConfig{Config: config, TokenFilePath: tokenFilePath},
		BindAddr:     "0.0.0.0",
		Port:         DefaultHTTPPort,
	}, nil
}

// Start starts the HTTP service
func (s HTTPService) Start() {
	l := logging.NewLogger("HTTPService.Start")
	l.Info().Msg("Starting HTTP server")
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", controllers.HealthHandler)
	router.GET("/google-validate", s.GoogleConfig.GoogleWebTokenHandler)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.BindAddr, s.Port),
		Handler: router,
	}

	app.RoutinesWG.Add(1)
	go s.server.ListenAndServe()
}

// Stop stops the HTTP service and free resources
func (s HTTPService) Stop() {
	l := logging.NewLogger("HTTPService.Stop")
	l.Info().Msg("Stopping HTTP server")
	if s.server != nil {
		s.server.Shutdown(context.TODO())
	}

	app.RoutinesWG.Done()
}
