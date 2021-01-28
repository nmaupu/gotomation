package smarthome

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
	Enabled  bool               `mapstructure:"enabled"`
	Entities []model.HassEntity `mapstructure:"trigger_entities"`
}

// IsEnabled godoc
func (a Action) IsEnabled() bool {
	return a.Enabled
}

// GetEntitiesForTrigger godoc
func (a Action) GetEntitiesForTrigger() []model.HassEntity {
	return a.Entities
}

// Trigger godoc
func (a Action) Trigger(e *model.HassEvent) {
	logging.Error("Action.Trigger").Err(errors.New("Not implemented")).Msg("")
}
