package triggers

import (
	"time"

	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ core.Actionable = (*Harmony)(nil)
)

// Harmony checks for harmony remote button press and takes action accordingly
type Harmony struct {
	core.Action `mapstructure:",squash"`
	WorkActions []workAction `mapstructure:"work_actions"`
}

type workAction struct {
	Key      string    `mapstructure:"key"`
	Commands []command `mapstructure:"commands"`
}

type command struct {
	// Entity to make action with
	Entity model.HassEntity `mapstructure:"entity"`
	// Service to call, basically turn_on, turn_off or toggle
	Service string `mapstructure:"service"`
	// Optional Delay to wait at the end of this action call
	Delay time.Duration `mapstructure:"delay"`
}

// Trigger godoc
func (h *Harmony) Trigger(event *model.HassEvent) {
	l := logging.NewLogger("Harmony.Trigger")

	if event == nil {
		l.Warn().Msg("Event received is nil")
		return
	}

	l.Trace().
		Str("event_type", event.Event.EventType).
		Str("data.source_name", event.Event.Data.SourceName).
		Str("data.type", event.Event.Data.Type).
		Str("data.key", event.Event.Data.Key).
		Msg("Trigger receiver for Harmony")

	cmds := h.getCommands(event.Event.Data.Key)
	for _, cmd := range cmds {
		cmdLogger := l.With().
			Str("cmd_entity", cmd.Entity.GetEntityIDFullName()).
			Str("cmd_service", cmd.Service).
			Str("cmd_delay", cmd.Delay.String()).
			Logger()
		cmdLogger.Debug().Msg("Calling service")
		if cmd.Entity.EntityID != "" && cmd.Entity.Domain != "" && cmd.Service != "" {
			err := httpclient.SimpleClientSingleton.CallService(cmd.Entity, cmd.Service)
			if err != nil {
				cmdLogger.Error().Err(err).Msg("An error occurred calling service")
			}
		}
		time.Sleep(cmd.Delay)
	}

}

func (h *Harmony) getCommands(key string) []command {
	for _, wa := range h.WorkActions {
		if key == wa.Key {
			return wa.Commands
		}
	}

	return []command{}
}
