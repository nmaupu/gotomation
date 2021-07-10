package smarthome

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/routines"
	"net/http"
	"time"
)

var (
	_ core.Actionable = (*RandomLightsTrigger)(nil)
)

const (
	offsetDefaultTimeBeginFromSunset = time.Minute * 30
	defaultNbSlots                   = 3
)

// RandomLightsTrigger checks for humidity and activate/deactivate a dehumidifier
type RandomLightsTrigger struct {
	core.Action `mapstructure:",squash"`
	// Name is the name of this RandomLightsTrigger
	Name string `mapstructure:"name"`
	// Lights are all the lights driven by the module
	Lights []model.RandomLight `mapstructure:"lights"`
	// NbSlots is the number maximum of lights set to on at the same time
	NbSlots uint32 `mapstructure:"nb_slots"`
	// TimeBegin is the starting time when the lights can be set to on
	TimeBegin time.Time `mapstructure:"time_begin"`
	// TimeEnd is the ending time when the lights can be set to on
	TimeEnd time.Time `mapstructure:"time_end"`

	randomLightsRoutine core.RandomLightsRoutine
}

// Trigger godoc
func (d *RandomLightsTrigger) Trigger(event *model.HassEvent) {
	var err error
	l := logging.NewLogger("RandomLightsTrigger.Trigger")
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
		if err := d.init(); err != nil {
			l.Error().Err(err).Msg("Unable to initialize")
			return
		}
		event.Event.Data.NewState.State = triggerEntity.State.String()
	}

	if event.Event.Data.NewState.IsON() {
		l.Debug().EmbedObject(triggerEntity).Msgf("Starting randomLightsRoutine")
		d.randomLightsRoutine.Start()
	} else {
		l.Debug().EmbedObject(triggerEntity).Msgf("Stopping randomLightsRoutine")
		d.randomLightsRoutine.Stop()
	}
}

func (d *RandomLightsTrigger) init() error {
	var err error

	if d.TimeBegin.IsZero() {
		_, sunset, err := core.Coords().GetSunriseSunset()
		if err != nil {
			return err
		}
		d.TimeBegin = sunset.Add(offsetDefaultTimeBeginFromSunset)
	}

	if d.TimeEnd.IsZero() {
		return fmt.Errorf("time_end is not specified")
	}

	if d.NbSlots == 0 {
		d.NbSlots = defaultNbSlots
	}

	d.randomLightsRoutine, err = core.NewRandomLightsRoutine(d.Name, d.NbSlots, d.Lights, d.TimeBegin, d.TimeEnd)
	routines.AddRunnable(d.randomLightsRoutine)
	return err
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
