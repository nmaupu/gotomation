package smarthome

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/nmaupu/gotomation/logging"
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

	logging.Trace("Trigger.Configure").Msgf("%+v", action)

	return nil
}

// GetActionable godoc
func (t *Trigger) GetActionable() Actionable {
	return t.Action
}
