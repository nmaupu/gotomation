package core

import (
	"fmt"

	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model/config"
)

var (
	_ Triggerable = (*Trigger)(nil)
)

// Trigger triggers an action when a change occurs
type Trigger struct {
	Action Actionable
}

// Configure godoc
func (t *Trigger) Configure(data interface{}, action interface{}) error {
	l := logging.NewLogger("Trigger.Configure")

	var ok bool
	t.Action, ok = action.(Actionable)
	if !ok {
		return fmt.Errorf("Cannot parse Actionable parameter")
	}

	err := config.NewMapStructureDecoder(t.Action).Decode(data)
	if err != nil {
		return err
	}

	l.Trace().Msgf("%+v", action)

	return nil
}

// GetActionable godoc
func (t *Trigger) GetActionable() Actionable {
	return t.Action
}

// GetName godoc
func (t *Trigger) GetName() string {
	return fmt.Sprintf("trigger/%s", t.Action.GetName())
}
