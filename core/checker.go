package core

import (
	"fmt"
	"time"

	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model/config"
)

var (
	logger           = logging.NewLogger("checker")
	_      Checkable = (*Checker)(nil)
)

// Checker checks a Modular at a regular interval
type Checker struct {
	stop   chan bool
	Module Modular
}

// Start starts to check
func (c *Checker) Start() error {
	if c.Module == nil {
		return fmt.Errorf("Checker is not configured: module is nil")
	}

	c.stop = make(chan bool, 1)

	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		ticker := time.NewTicker(c.Module.GetInterval())
		defer ticker.Stop()

		// Ensure executing module's check right away before first tick
		// https://github.com/golang/go/issues/17601
		if c.Module.IsEnabled() {
			select {
			case <-c.stop:
				return
			default:
				c.Module.Check()
			}
		}

		for c.Module.IsEnabled() {
			select {
			case <-c.stop:
				return
			case <-ticker.C:
				c.Module.Check()
			}
		}
	}()

	return nil
}

// Stop stops to check
func (c *Checker) Stop() {
	c.stop <- true
}

// Configure reads the configuration and returns a new Checkable object
func (c *Checker) Configure(data interface{}, module interface{}) error {
	l := logging.NewLogger("checker.Configure")

	var ok bool
	c.Module, ok = module.(Modular)
	if !ok {
		return fmt.Errorf("Cannot parse Modular parameter")
	}

	err := config.NewMapStructureDecoder(c.Module).Decode(data)
	if err != nil {
		return err
	}

	l.Trace().
		Str("module", fmt.Sprintf("%+v", module))

	return nil
}

// GetName returns the name of this runnable object
func (c *Checker) GetName() string {
	return fmt.Sprintf("Checker/%s", c.Module.GetName())
}
