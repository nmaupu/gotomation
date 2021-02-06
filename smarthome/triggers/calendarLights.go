package triggers

import (
	"github.com/nmaupu/gotomation/core"
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
	l := logging.NewLogger("CalendarLights.Trigger")

	l.Debug().Str("event_entity_id", event.Event.Data.EntityID).Msg("Trigger called")
}
