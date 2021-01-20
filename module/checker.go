package module

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/prometheus/common/log"
)

var (
	_ Checkable = (*Checker)(nil)
)

// Checker calls Check func every Interval duration
// A Checker defines default funcs: Start, Stop and Configure
// As Go lacks funcs' override capability, we need to pass the destination implementation (which has the real Check func implemented)
// first to Configure and to Check
//
// Example:
// checker := new(InternetChecker)
// checker.Configure(data, checker)
// checker.Start()
//
// We need to pass the checker object back into Configure because "this" refers to this Checker struct whereas we need to
// store the object into InternetChecker struct (also implenting the Checkable interface)
type Checker struct {
	Enabled  bool          `mapstructure:"enabled"`
	Interval time.Duration `mapstructure:"interval"`
	stop     chan bool
	module   Module
	destImpl Checkable
}

// Start starts to check
func (c *Checker) Start(checkFunc func()) {
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
				if c.destImpl != nil {
					c.destImpl.Check()
				} else {
					c.Check()
				}
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
	log.Errorf("Check is not defined, nothing to do")
}

// Configure reads the configuration and returns a new Checkable object
func (c *Checker) Configure(data interface{}, destImpl Checkable) error {
	c.destImpl = destImpl
	mapstructureConfig := &mapstructure.DecoderConfig{
		DecodeHook: MapstructureDecodeHook,
		Result:     c.destImpl,
	}
	decoder, _ := mapstructure.NewDecoder(mapstructureConfig)
	err := decoder.Decode(data)
	if err != nil {
		return err
	}

	fmt.Printf("%+v", destImpl)

	return nil
}
