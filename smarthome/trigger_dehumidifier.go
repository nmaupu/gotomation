package smarthome

import (
	"strconv"
	"time"

	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ Actionable = (*DehumidifierTrigger)(nil)
)

// DehumidifierTrigger checks for humidity and activate/deactivate a dehumidifier
type DehumidifierTrigger struct {
	Action `mapstructure:",squash"`
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
func (t *DehumidifierTrigger) Trigger(event *model.HassEvent) {
	if event == nil {
		return
	}

	switch event.Event.Data.EntityID {
	case t.ManualOverride.GetEntityIDFullName():
		logging.Debug("DehumidifierTrigger.Trigger").
			Str("state", event.Event.Data.NewState.State).
			Msg("Manual override changed")

	default:
		currentHum, err := strconv.ParseFloat(event.Event.Data.NewState.State, 64)
		if err != nil {
			logging.Error("DehumidifierTrigger.Trigger").Err(err).Msg("Error parsing humidity")
			return // Should not happen
		}

		if !t.inTimeRange() {
			logging.Debug("DehumidifierTrigger.Trigger").
				Float64("current", currentHum).
				Float64("threshold_min", t.ThresholdMin).
				Float64("threshold_max", t.ThresholdMax).
				Str("time_beg", t.TimeBeg.Format("15:04:05")).
				Str("time_end", t.TimeEnd.Format("15:04:05")).
				Msg("Current time is not in range, nothing to do")
			return
		}

		logging.Trace("DehumidifierTrigger.Trigger").
			EmbedObject(event).
			Msg("Event received")

		switchState, err := httpclient.SimpleClientSingleton.GetEntity(t.SwitchEntity.Domain, t.SwitchEntity.EntityID)
		if err != nil {
			logging.Error("DehumidifierTrigger.Trigger").Err(err).
				Str("device", t.SwitchEntity.GetEntityIDFullName()).
				Msg("Error, unable to get state for device")
		}

		if currentHum >= t.ThresholdMax {
			// in range or superior to ThresholdMax - ensure on
			if switchState.State.State == "off" {
				logging.Debug("DehumidifierTrigger.Trigger").
					Float64("current", currentHum).
					Float64("threshold_min", t.ThresholdMin).
					Float64("threshold_max", t.ThresholdMax).
					Msg("current >= threshold_max, switching on")
				httpclient.SimpleClientSingleton.CallService(t.SwitchEntity, "turn_on")
			} else {
				logging.Debug("DehumidifierTrigger.Trigger").
					Float64("current", currentHum).
					Float64("threshold_min", t.ThresholdMin).
					Float64("threshold_max", t.ThresholdMax).
					Msg("current >= threshold_max but already on, doing nothing")
			}
		} else if currentHum <= t.ThresholdMin {
			// in range or superior to ThresholdMax - ensure on
			if switchState.State.State == "on" {
				logging.Debug("DehumidifierTrigger.Trigger").
					Float64("current", currentHum).
					Float64("threshold_min", t.ThresholdMin).
					Float64("threshold_max", t.ThresholdMax).
					Msg("current <= threshold_min, switching off")
				httpclient.SimpleClientSingleton.CallService(t.SwitchEntity, "turn_off")
			} else {
				logging.Debug("DehumidifierTrigger.Trigger").
					Float64("current", currentHum).
					Float64("threshold_min", t.ThresholdMin).
					Float64("threshold_max", t.ThresholdMax).
					Msg("current <= threshold_min but already off, doing nothing")
			}
		} else {
			logging.Debug("DehumidifierTrigger.Trigger").
				Float64("current", currentHum).
				Float64("threshold_min", t.ThresholdMin).
				Float64("threshold_max", t.ThresholdMax).
				Msg("Nothing to do")
		}
	}

}

// inTimeRange checks if current time is in between TimeBeg and TimeEnd
func (t *DehumidifierTrigger) inTimeRange() bool {
	now := time.Now().Local()
	beg := time.Date(now.Year(), now.Month(), now.Day(), t.TimeBeg.Hour(), t.TimeBeg.Minute(), t.TimeBeg.Second(), 0, time.Local)
	end := time.Date(now.Year(), now.Month(), now.Day(), t.TimeEnd.Hour(), t.TimeEnd.Minute(), t.TimeEnd.Second(), 0, time.Local)
	return now.After(beg) && now.Before(end)
}
