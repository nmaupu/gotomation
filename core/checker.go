package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model/config"
)

const (
	// DefaultInterval is used when interval is not set (or set to zero)
	DefaultInterval = time.Minute * 10
)

var (
	logger           = logging.NewLogger("checker")
	_      Checkable = (*Checker)(nil)
)

// Checker checks a Modular at a regular interval
type Checker struct {
	stop   chan bool
	Module Modular

	started        bool
	mutexStopStart sync.Mutex
}

// Start starts to check
func (c *Checker) Start() error {
	c.mutexStopStart.Lock()
	defer c.mutexStopStart.Unlock()
	if c.started {
		return nil
	}

	if c.Module == nil {
		return fmt.Errorf("Checker is not configured: module is nil")
	}

	c.stop = make(chan bool, 1)

	app.RoutinesWG.Add(1)
	go func() {
		l := logging.NewLogger("Checker.Start")
		defer app.RoutinesWG.Done()
		interval := DefaultInterval
		if c.Module.GetInterval() > 0 {
			interval = c.Module.GetInterval()
		}
		ticker := time.NewTicker(interval)
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

		for {
			select {
			case <-c.stop:
				return
			case <-ticker.C:
				if c.Module.IsEnabled() {
					l.Trace().Msgf("Checker %s is enabled, calling Check()", c.Module.GetName())
					c.Module.Check()
				} else {
					l.Trace().Msgf("Checker %s is disabled, doing nothing", c.Module.GetName())
				}
			}
		}
	}()

	c.started = true
	return nil
}

// Stop stops to check
func (c *Checker) Stop() {
	c.mutexStopStart.Lock()
	defer c.mutexStopStart.Unlock()
	if !c.started {
		return
	}

	c.stop <- true
	c.started = false
}

// IsStarted checks whether or not the routine is already started
func (c *Checker) IsStarted() bool {
	c.mutexStopStart.Lock()
	defer c.mutexStopStart.Unlock()
	return c.started
}

// Configure reads the configuration and returns a new Checkable object
func (c *Checker) Configure(data interface{}, module interface{}) error {
	l := logging.NewLogger("checker.Configure")

	var ok bool
	c.Module, ok = module.(Modular)
	if !ok {
		return fmt.Errorf("cannot parse Modular parameter")
	}

	err := config.NewMapstructureDecoder(c.Module).Decode(data)
	if err != nil {
		return err
	}

	l.Trace().
		Str("module", fmt.Sprintf("%+v", module))

	return nil
}

// GetName returns the name of this runnable object
func (c *Checker) GetName() string {
	return fmt.Sprintf("checker/%s", c.Module.GetName())
}

// GetModular godoc
func (c *Checker) GetModular() Modular {
	return c.Module
}
