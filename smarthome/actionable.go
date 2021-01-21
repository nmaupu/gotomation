package smarthome

import "github.com/nmaupu/gotomation/model"

// Actionable is an interface to react on change event
type Actionable interface {
	IsEnabled() bool
	GetEntitiesForTrigger() []model.HassEntity
	Trigger(e *model.HassEvent)
}
