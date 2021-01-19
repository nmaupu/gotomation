package module

import (
	"errors"
	"log"
	"time"
)

var (
	_ Checkable = (*Checker)(nil)
)

// Checker calls Executor every Interval duration
type Checker struct {
	Enabled  bool
	Interval time.Duration
	stop     chan bool
	Checker  Checkable
}

// Start starts to check
func (c *Checker) Start() {
	c.stop = make(chan bool, 1)

	go func() {
		ticker := time.NewTicker(c.Interval)
		defer ticker.Stop()

		for c.Enabled {
			select {
			case s := <-c.stop:
				if s {
					return
				}
			case <-ticker.C:
				c.Checker.Check()
			}
		}
	}()
}

// Stop stops to check
func (c *Checker) Stop() {
	c.stop <- true
}

// IsEnabled returns true if the module is enabled, false otherwise
func (c *Checker) IsEnabled() bool {
	return c.Enabled
}

// Check godoc
func (c *Checker) Check() {
	log.Println("Check func not implemented")
}

// Configure godoc
func (c *Checker) Configure(data interface{}) error {
	return errors.New("Configure func not implemented")
}
