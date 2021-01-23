package smarthome

import (
	"fmt"
	"log"

	"github.com/mitchellh/mapstructure"
)

var (
	_ Triggerable = (*Trigger)(nil)
)

// Trigger triggers an action when a change occurs
type Trigger struct {
	Action Actionable
}

// Configure godoc
func (t *Trigger) Configure(config interface{}, action interface{}) error {
	var ok bool
	t.Action, ok = action.(Actionable)
	if !ok {
		return fmt.Errorf("Cannot parse Actionable parameter")
	}

	mapstructureConfig := &mapstructure.DecoderConfig{
		DecodeHook: MapstructureDecodeHook,
		Result:     t.Action,
	}
	decoder, _ := mapstructure.NewDecoder(mapstructureConfig)
	err := decoder.Decode(config)
	if err != nil {
		return err
	}

	log.Printf("%+v\n", action)

	return nil
}

// GetActionable godoc
func (t *Trigger) GetActionable() Actionable {
	return t.Action
}
