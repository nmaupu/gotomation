package triggers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ (core.Actionable) = (*CalendarLights)(nil)
)

// CalendarLights checks calendar for new events once in a while
type CalendarLights struct {
	core.Action `mapstructure:",squash"`
	Cals        []struct {
		Name string `mapstructure:"name"`
		ID   string `mapstructure:"id"`
	} `mapstructure:"cals"`
}

// Trigger godoc
func (c *CalendarLights) Trigger(event *model.HassEvent) {
	var err error
	l := logging.NewLogger("CalendarLights.Trigger").With().EmbedObject(event).Logger()

	if !event.OppositeState() {
		l.Debug().Msg("Old and new states are not opposite, ignoring event")
		return
	}

	service := fmt.Sprintf("turn_%s", strings.ToLower(event.Event.Data.NewState.State))

	// Getting extras parameters from calendar's event description
	extraParamsJSON := event.Event.Data.NewState.Attributes["description"].(string)
	extraParams := make(map[string]interface{}, 0)
	if event.Event.Data.NewState.State == model.StateON {
		err = json.Unmarshal([]byte(extraParamsJSON), &extraParams)
		if err != nil {
			l.Error().Err(err).Msg("Unable to unmarshal calendar's description for extra parameters")
			return
		}
	}

	// Looking for real light entity
	eventEntity := model.NewHassEntity(event.Event.Data.EntityID) // Should get calendar.light_xxx
	lightEntity := model.NewHassEntity(strings.Replace(eventEntity.EntityID, "_", ".", 1))
	entity, err := httpclient.GetSimpleClient().GetEntity(lightEntity.Domain, fmt.Sprintf("%s.*", lightEntity.EntityID))
	if err != nil {
		l.Error().Err(err).EmbedObject(lightEntity).Msg("Unable to get entity")
		return
	}

	// Switching entity on or off
	err = httpclient.GetSimpleClient().CallService(entity, service, extraParams)
	if err != nil {
		l.Error().Err(err).EmbedObject(entity).Msgf("Cannot call service %s on entity", service)
	}
}

// GinHandler godoc
func (c *CalendarLights) GinHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, *c)
}
