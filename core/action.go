package core

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ Actionable = (*Action)(nil)
)

// Action is triggered on state change
type Action struct {
	automate `mapstructure:",squash"`
	// Entities triggers events only from specific entities. No filter -> all events arrive
	// Use either trigger_entities OR trigger_events
	Entities []model.HassEntity `mapstructure:"trigger_entities"`
	// EventTypes triggers events only from specific event type
	// Use either trigger_entities OR trigger_events
	EventTypes []string `mapstructure:"trigger_events"`
}

// GetEntitiesForTrigger godoc
func (a *Action) GetEntitiesForTrigger() []model.HassEntity {
	return a.Entities
}

// GetEventTypesForTrigger godoc
func (a *Action) GetEventTypesForTrigger() []string {
	return a.EventTypes
}

// Trigger godoc
func (a *Action) Trigger(e *model.HassEvent) {
	l := logging.NewLogger("Action.Trigger")
	l.Error().Err(errors.New("not implemented")).Msg("")
}

// GinHandler godoc
func (a *Action) GinHandler(c *gin.Context) {
	c.JSON(http.StatusOK, a)
}

// GetName godoc
func (a *Action) GetName() string {
	return a.Name
}

// NeedsInitialization specifies if this Actionable needs to be triggered with a dummy event
// when program starts (or conf is reloaded)
func (a *Action) NeedsInitialization() bool {
	return false
}
