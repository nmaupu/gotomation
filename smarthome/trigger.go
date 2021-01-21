package smarthome

import (
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
func (t *Trigger) Configure(data interface{}, action Actionable) error {
	t.Action = action
	mapstructureConfig := &mapstructure.DecoderConfig{
		DecodeHook: MapstructureDecodeHook,
		Result:     t.Action,
	}
	decoder, _ := mapstructure.NewDecoder(mapstructureConfig)
	err := decoder.Decode(data)
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
