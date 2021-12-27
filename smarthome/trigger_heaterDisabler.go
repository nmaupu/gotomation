package smarthome

import (
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

// HeaterCheckersDisablerTrigger globally disables all heaters (or specified ones) and set a default temperature
type HeaterCheckersDisablerTrigger struct {
	core.Action `mapstructure:",squash"`
}

// Trigger godoc
func (d *HeaterCheckersDisablerTrigger) Trigger(event *model.HassEvent) {
	l := logging.NewLogger("HeaterCheckersDisabler.Trigger")

	l.Trace().
		EmbedObject(event).
		Msg("Trigger event occurred")
	d.setAllCheckers(event.Event.Data.NewState.IsON())
}

func (d *HeaterCheckersDisablerTrigger) setAllCheckers(overrideState bool) {
	l := logging.NewLogger("HeaterCheckersDisablerTrigger.setAllCheckers").With().Bool("overrideState", overrideState).Logger()

	for _, checker := range GetCheckersByType(ModuleHeaterChecker) {
		m := checker.GetModular()
		heaterChecker, ok := m.(*HeaterChecker)
		if !ok {
			continue
		}

		turnOffManualOverride := func() {
			manualOverrideEntity, err := heaterChecker.GetManualOverrideEntity()
			if err != nil {
				l.Error().Err(err).
					Object("entity", manualOverrideEntity).
					Msg("Unable to get manual override entity")
			} else {
				// Turn off manual override for this heater
				err := httpclient.GetSimpleClient().CallService(manualOverrideEntity, "turn_off", map[string]interface{}{})
				if err != nil {
					l.Error().Err(err).
						Object("entity", manualOverrideEntity).
						Msg("Cannot turn off manual override")
				}
			}
		}

		if overrideState {
			l.Info().Str("heater_checker", heaterChecker.Name).Msg("Disabling heater's checker")
			heaterChecker.Module.Disable()

			turnOffManualOverride()

			// Set temp to default eco
			temp := heaterChecker.GetDefaultEcoTemp()
			climateEntity, err := heaterChecker.GetClimateEntity()
			if err != nil {
				l.Error().Err(err).
					Object("entity", climateEntity).
					Msg("Unable to get climate entity")
			} else {
				err := httpclient.GetSimpleClient().CallService(climateEntity, "set_temperature", map[string]interface{}{
					"temperature": temp,
				})
				if err != nil {
					l.Error().Err(err).
						Object("entity", climateEntity).
						Float64("temperature", temp).
						Msg("Cannot set climate's temperature")
				}
			}
		} else {
			l.Info().Str("heater_checker", heaterChecker.Name).Msg("Enabling heater's checker")
			heaterChecker.Module.Enable()
			turnOffManualOverride()
			// next check call will reset the correct temperature
		}
	}
}
