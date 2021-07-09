package core

import (
	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/model"
)

// Actionable is an interface to react on change event
type Actionable interface {
	Automate
	GetEntitiesForTrigger() []model.HassEntity
	GetEventTypesForTrigger() []string
	Trigger(e *model.HassEvent)
	GinHandler(c *gin.Context)
	// NeedsInitialization specifies if this Actionable needs to be triggered with a dummy event
	// when program starts (or conf is reloaded)
	NeedsInitialization() bool
}
