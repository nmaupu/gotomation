package core

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/logging"
)

var (
	_ Modular = (*Module)(nil)
)

// Module is the base struct to build a module
type Module struct {
	automate `mapstructure:",squash"`
	Interval time.Duration `mapstructure:"interval"`
}

// Check godoc
func (m *Module) Check() {
	l := logging.NewLogger("Module.Check")
	l.Error().Err(errors.New("not implemented")).Send()
}

// GetInterval godoc
func (m *Module) GetInterval() time.Duration {
	return m.Interval
}

// GinHandler godoc
func (m *Module) GinHandler(c *gin.Context) {
	c.JSON(http.StatusOK, m)
}
