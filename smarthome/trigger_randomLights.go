package smarthome

import (
	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"net/http"
)

var (
	_ core.Actionable = (*RandomLightsTrigger)(nil)
)

// RandomLightsTrigger checks for humidity and activate/deactivate a dehumidifier
type RandomLightsTrigger struct {
	core.Action `mapstructure:",squash"`
}

// Trigger godoc
func (d *RandomLightsTrigger) Trigger(event *model.HassEvent) {
	var err error
	l := logging.NewLogger("DehumidifierTrigger.Trigger")
	if event == nil {
		return
	}

	if len(d.Entities) != 1 {
		l.Warn().Msg("Unable to trigger RandomLightsTrigger because there is more than one trigger entity")
	}

	// Initialization depending on input_boolean status
	triggerEntity, err := httpclient.GetSimpleClient().GetEntity(d.Entities[0].Domain, d.Entities[0].EntityID)
	if err != nil {
		l.Error().Err(err).EmbedObject(triggerEntity).Msg("unable to get entity's state")
		return
	}

	// initializing event state to entity's state
	if event.IsDummy() {
		event.Event.Data.NewState.State = triggerEntity.State.String()
	}

	l.Debug().EmbedObject(triggerEntity).Msgf("Entity status state = %s", event.Event.Data.NewState.State)
}

// GinHandler godoc
func (d *RandomLightsTrigger) GinHandler(c *gin.Context) {
	c.JSON(http.StatusOK, d)
}

// NeedsInitialization specifies if this Actionable needs to be triggered with a dummy event
// when program starts (or conf is reloaded)
func (d *RandomLightsTrigger) NeedsInitialization() bool {
	return true
}
