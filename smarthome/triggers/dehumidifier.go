package triggers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ core.Actionable = (*Dehumidifier)(nil)
)

// Dehumidifier checks for humidity and activate/deactivate a dehumidifier
type Dehumidifier struct {
	core.Action `mapstructure:",squash"`
	// SwitchEntity is the entity used to switch on / off the dehumidifier
	SwitchEntity model.HassEntity `mapstructure:"switch_entity"`
	// TimeBeg is the time where monitoring begins
	TimeBeg time.Time `mapstructure:"time_beg"`
	// TimeEnd is the time where monitoring ends
	TimeEnd time.Time `mapstructure:"time_end"`
	// ThresholdMin is the threshold which drives the dehumidifier on/off
	ThresholdMin float64 `mapstructure:"threshold_min"`
	// ThresholdMax is the threshold which drives the dehumidifier on/off
	ThresholdMax float64 `mapstructure:"threshold_max"`
	// ManualOverride is the input_boolean to deactivate manually the DehumidifierChecker automatic behavior
	ManualOverride model.HassEntity `mapstructure:"manual_override"`
}

// Trigger godoc
func (d *Dehumidifier) Trigger(event *model.HassEvent) {
	l := logging.NewLogger("DehumidifierTrigger.Trigger")
	if event == nil {
		return
	}

	switch event.Event.Data.EntityID {
	case d.ManualOverride.GetEntityIDFullName():
		// TODO implement manual override change
		l.Debug().
			Str("state", event.Event.Data.NewState.State).
			Msg("Manual override changed (not implemented)")

	default:
		currentHum, err := strconv.ParseFloat(event.Event.Data.NewState.State, 64)
		if err != nil {
			l.Error().Err(err).Msg("Error parsing humidity")
			return // Should not happen
		}

		l.Trace().
			EmbedObject(event).
			Msg("Event received")

		switchState, err := httpclient.GetSimpleClient().GetEntity(d.SwitchEntity.Domain, d.SwitchEntity.EntityID)
		if err != nil {
			l.Error().Err(err).
				Str("device", d.SwitchEntity.GetEntityIDFullName()).
				Msg("Error, unable to get state for device")
		}

		if !d.inTimeRange() && switchState.State.State != "on" { // If not in time range, let it get under threshold before switching off
			l.Debug().
				Float64("current", currentHum).
				Float64("threshold_min", d.ThresholdMin).
				Float64("threshold_max", d.ThresholdMax).
				Str("time_beg", d.TimeBeg.Format("15:04:05")).
				Str("time_end", d.TimeEnd.Format("15:04:05")).
				Msg("Current time is not in range, nothing to do")
			return
		}

		if currentHum >= d.ThresholdMax {
			// in range or superior to ThresholdMax - ensure on
			if switchState.State.State == "off" {
				l.Debug().
					Float64("current", currentHum).
					Float64("threshold_min", d.ThresholdMin).
					Float64("threshold_max", d.ThresholdMax).
					Msg("current >= threshold_max, switching on")
				httpclient.GetSimpleClient().CallService(d.SwitchEntity, "turn_on", nil)
			} else {
				l.Debug().
					Float64("current", currentHum).
					Float64("threshold_min", d.ThresholdMin).
					Float64("threshold_max", d.ThresholdMax).
					Msg("current >= threshold_max but already on, doing nothing")
			}
		} else if currentHum <= d.ThresholdMin {
			// in range or superior to ThresholdMax - ensure on
			if switchState.State.State == "on" {
				l.Debug().
					Float64("current", currentHum).
					Float64("threshold_min", d.ThresholdMin).
					Float64("threshold_max", d.ThresholdMax).
					Msg("current <= threshold_min, switching off")
				httpclient.GetSimpleClient().CallService(d.SwitchEntity, "turn_off", nil)
			} else {
				l.Debug().
					Float64("current", currentHum).
					Float64("threshold_min", d.ThresholdMin).
					Float64("threshold_max", d.ThresholdMax).
					Msg("current <= threshold_min but already off, doing nothing")
			}
		} else {
			l.Debug().
				Float64("current", currentHum).
				Float64("threshold_min", d.ThresholdMin).
				Float64("threshold_max", d.ThresholdMax).
				Msg("Nothing to do")
		}
	}

}

// inTimeRange checks if current time is between TimeBeg and TimeEnd
func (d *Dehumidifier) inTimeRange() bool {
	now := time.Now()
	beg := time.Date(now.Year(), now.Month(), now.Day(), d.TimeBeg.Hour(), d.TimeBeg.Minute(), d.TimeBeg.Second(), 0, time.Local)
	end := time.Date(now.Year(), now.Month(), now.Day(), d.TimeEnd.Hour(), d.TimeEnd.Minute(), d.TimeEnd.Second(), 0, time.Local)
	return now.After(beg) && now.Before(end)
}

// GinHandler godoc
func (d *Dehumidifier) GinHandler(c *gin.Context) {
	c.JSON(http.StatusOK, d)
}
