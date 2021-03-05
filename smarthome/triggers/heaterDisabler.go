package triggers

import (
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

// HeaterCheckersDisabler globally disables all heaters (or specified ones) and set a default temperature
type HeaterCheckersDisabler struct {
	core.Action  `mapstructure:",squash"`
	Temp         float64  `mapstructure:"temp"`
	CheckerNames []string `mapstructure:"checkers"`
}

// Trigger godoc
func (d *HeaterCheckersDisabler) Trigger(event *model.HassEvent) {
	l := logging.NewLogger("HeaterCheckersDisabler")

	l.Info().
		EmbedObject(event).
		Msg("Trigger event occurred")
}
