package core

import (
	"errors"

	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ Actionable = (*Action)(nil)
)

// Action is triggered on state change
type Action struct {
	// Enabled enables or disables the Action object
	Enabled bool `mapstructure:"enabled"`
	// Entities triggers events only from specific entities. No filter -> all events arrive
	// Use either trigger_entities OR trigger_events
	Entities []model.HassEntity `mapstructure:"trigger_entities"`
	// EventTypes triggers events only from specific event type
	// Use either trigger_entities OR trigger_events
	EventTypes []string `mapstructure:"trigger_events"`
}

// IsEnabled godoc
func (a Action) IsEnabled() bool {
	return a.Enabled
}

// GetEntitiesForTrigger godoc
func (a Action) GetEntitiesForTrigger() []model.HassEntity {
	return a.Entities
}

// GetEventTypesForTrigger godoc
func (a Action) GetEventTypesForTrigger() []string {
	return a.EventTypes
}

// Trigger godoc
func (a Action) Trigger(e *model.HassEvent) {
	l := logging.NewLogger("Action.Trigger")
	l.Error().Err(errors.New("Not implemented")).Msg("")
}
