package core

import (
	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/model"
)

// Actionable is an interface to react on change event
type Actionable interface {
	IsEnabled() bool
	GetName() string
	GetEntitiesForTrigger() []model.HassEntity
	GetEventTypesForTrigger() []string
	Trigger(e *model.HassEvent)
	GinHandler(c *gin.Context)
}
