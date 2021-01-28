package core

import (
	"errors"
	"time"

	"github.com/nmaupu/gotomation/logging"
)

var (
	_ (Modular) = (*Module)(nil)
)

// Module is the base struct to build a module
type Module struct {
	Enabled  bool          `mapstructure:"enabled"`
	Interval time.Duration `mapstructure:"interval"`
}

// Check godoc
func (m Module) Check() {
	logging.Error("Module.Check").Err(errors.New("Not implemented")).Msg("")
}

// GetInterval godoc
func (m Module) GetInterval() time.Duration {
	return m.Interval
}

// IsEnabled godoc
func (m Module) IsEnabled() bool {
	return m.Enabled
}
