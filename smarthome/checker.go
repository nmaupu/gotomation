package smarthome

import (
	"fmt"
	"log"
	"time"

	"github.com/mitchellh/mapstructure"
)

var (
	_ Checkable = (*Checker)(nil)
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

	go func() {
		ticker := time.NewTicker(c.Module.GetInterval())
		defer ticker.Stop()

		for c.Module.IsEnabled() {
			select {
			case s := <-c.stop:
				if s {
					return
				}
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
func (c *Checker) Configure(data interface{}, module Modular) error {
	c.Module = module
	mapstructureConfig := &mapstructure.DecoderConfig{
		DecodeHook: MapstructureDecodeHook,
		Result:     c.Module,
	}
	decoder, _ := mapstructure.NewDecoder(mapstructureConfig)
	err := decoder.Decode(data)
	if err != nil {
		return err
	}

	log.Printf("%+v\n", module)

	return nil
}