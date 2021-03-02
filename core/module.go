package core

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/logging"
)

var (
	_ (Modular) = (*Module)(nil)
)

// Module is the base struct to build a module
type Module struct {
	Name     string        `mapstructure:"name"`
	Enabled  bool          `mapstructure:"enabled"`
	Interval time.Duration `mapstructure:"interval"`
}

// Check godoc
func (m Module) Check() {
	l := logging.NewLogger("Module.Check")
	l.Error().Err(errors.New("Not implemented")).Send()
}

// GetInterval godoc
func (m Module) GetInterval() time.Duration {
	return m.Interval
}

// IsEnabled godoc
func (m Module) IsEnabled() bool {
	return m.Enabled
}

// GetName return the name of the module
func (m Module) GetName() string {
	return m.Name
}

// GinHandler godoc
func (m Module) GinHandler(c *gin.Context) {
	c.JSON(http.StatusOK, m)
}
