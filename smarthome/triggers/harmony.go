package triggers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ core.Actionable = (*Harmony)(nil)
)

const (
	offsetDawn = 15 * time.Minute
	offsetDusk = -15 * time.Minute
)

// Harmony checks for harmony remote button press and takes action accordingly
type Harmony struct {
	core.Action `mapstructure:",squash"`
	WorkActions []workAction `mapstructure:"work_actions"`
}

type workAction struct {
	// Key is the key pressed to trigger this workAction
	Key string `mapstructure:"key"`
	// OnlyDark triggers this workAction only when it's dark outside
	OnlyDark bool `mapstructure:"only_dark"`
	// Commands are all the commands being executed
	Commands []command `mapstructure:"commands"`
}

type command struct {
	// Entity to make action with
	Entity model.HassEntity `mapstructure:"entity"`
	// Service to call, basically turn_on, turn_off or toggle
	Service string `mapstructure:"service"`
	// Optional Delay to wait at the end of this action call
	Delay time.Duration `mapstructure:"delay"`
	// Brightness is the brightness to set (for compatible device)
	Brightness int `mapstructure:"brightness"`
}

// Trigger godoc
func (h *Harmony) Trigger(event *model.HassEvent) {
	l := logging.NewLogger("Harmony.Trigger")

	if event == nil {
		l.Warn().Msg("Event received is nil")
		return
	}

	l = l.With().
		Str("event_type", event.Event.EventType).
		Str("data.source_name", event.Event.Data.SourceName).
		Str("data.type", event.Event.Data.Type).
		Str("data.key", event.Event.Data.Key).
		Logger()

	l.Trace().Msg("Trigger receiver for Harmony")

	wa := h.getWorkAction(event.Event.Data.Key)
	if wa == nil {
		l.Warn().Msg("No action for this key")
		return
	}

	if !wa.OnlyDark || (wa.OnlyDark && core.Coords().IsDarkNow(offsetDawn, offsetDusk)) {
		for _, cmd := range wa.Commands {
			cmdLogger := l.With().
				Str("cmd_entity", cmd.Entity.GetEntityIDFullName()).
				Str("cmd_service", cmd.Service).
				Str("cmd_delay", cmd.Delay.String()).
				Int("cmd_brightness", cmd.Brightness).
				Logger()

			cmdLogger.Debug().Msg("Calling service")
			if cmd.Entity.EntityID != "" && cmd.Entity.Domain != "" && cmd.Service != "" {
				extra := make(map[string]interface{}, 0)
				if cmd.Brightness > 0 {
					extra["brightness"] = cmd.Brightness
				}
				err := httpclient.GetSimpleClient().CallService(cmd.Entity, cmd.Service, extra)
				if err != nil {
					cmdLogger.Error().Err(err).Msg("An error occurred calling service")
				}
			}
			time.Sleep(cmd.Delay)
		}
	} else {
		l.Debug().Msg("Not dark now, doing nothing")
	}
}

func (h *Harmony) getWorkAction(key string) *workAction {
	for _, wa := range h.WorkActions {
		if key == wa.Key {
			return &wa
		}
	}

	return nil
}

// GinHandler godoc
func (h *Harmony) GinHandler(c *gin.Context) {
	c.JSON(http.StatusOK, *h)
}
