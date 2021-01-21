package smarthome

import (
	"log"
	"time"

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
	ThresholdMin float32 `mapstructure:"threshold_min"`
	// ThresholdMax is the threshold which drives the dehumidifier on/off
	ThresholdMax float32 `mapstructure:"threshold_max"`
	// ManualOverride is the input_boolean to deactivate manually the DehumidifierChecker automatic behavior
	ManualOverride model.HassEntity `mapstructure:"manual_override"`
	// ManualOverrideReset is the time where the ManualOverride input_boolean is automatically deactivated
	ManualOverrideResetTime time.Time `mapstructure:"manual_override_reset_time"`
}

// Trigger godoc
func (t *DehumidifierTrigger) Trigger(event *model.HassEvent) {
	if event == nil {
		return
	}

	log.Printf("[DehumidifierTrigger] Received: event, msg=%+v\n", event)
}
