package smarthome

import (
	"log"
	"strconv"
	"time"

	"github.com/nmaupu/gotomation/httpclient"
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
		//log.Printf("[DehumidifierTrigger] Received: event, msg=%+v\n", event)
		log.Printf("Manual override state: %s", event.Event.Data.NewState.State)

	default:
		if !t.inTimeRange() {
			log.Printf("Current time is not between %s and %s, nothing to do", t.TimeBeg.Format("15:04:05"), t.TimeEnd.Format("15:04:05"))
			return
		}

		//log.Printf("[DehumidifierTrigger] Received: event, msg=%+v\n", event)
		currentHum, err := strconv.ParseFloat(event.Event.Data.NewState.State, 64)
		if err != nil {
			return // Should not happen
		}

		switchState, err := httpclient.SimpleClientSingleton.GetEntity(t.SwitchEntity.Domain, t.SwitchEntity.EntityID)
		if err != nil {
			log.Printf("[DehumidifierTrigger] Error, unable to get state for device %s, err=%v", t.SwitchEntity.GetEntityIDFullName(), err)
		}

		if currentHum >= t.ThresholdMax {
			// in range or superior to ThresholdMax - ensure on
			if switchState.State.State == "off" {
				log.Printf("[DehumidifierTrigger] %f >= %f, switching on", currentHum, t.ThresholdMax)
				httpclient.SimpleClientSingleton.CallService(t.SwitchEntity, "turn_on")
			} else {
				log.Printf("[DehumidifierTrigger] %f >= %f, already on, doing nothing", currentHum, t.ThresholdMax)
			}
		} else if currentHum <= t.ThresholdMin {
			// in range or superior to ThresholdMax - ensure on
			if switchState.State.State == "on" {
				log.Printf("[DehumidifierTrigger] %f <= %f, switching off", currentHum, t.ThresholdMin)
				httpclient.SimpleClientSingleton.CallService(t.SwitchEntity, "turn_off")
			} else {
				log.Printf("[DehumidifierTrigger] %f <= %f, already off, doing nothing", currentHum, t.ThresholdMin)
			}
		} else {
			log.Printf("[DehumidifierTrigger] current_hum=%f, threshold_min=%f, threshold_max=%f, nothing to do", currentHum, t.ThresholdMin, t.ThresholdMax)
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
