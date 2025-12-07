package smarthome

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/routines"
)

const (
	// DefaultEcoTemp is the default eco temperature when not set in the config file
	DefaultEcoTemp           = float64(15)
	temperatureAttributeName = "temperature"
	setTemperatureService    = "set_temperature"
	climateTurnOffService    = "turn_off"
	climateTurnOnService     = "turn_on"
)

var (
	_ core.Modular = (*HeaterChecker)(nil)
)

// HeaterChecker sets the heater's thermostat based on schedules
type HeaterChecker struct {
	core.Module   `mapstructure:",squash"`
	SchedulesFile string `mapstructure:"schedules_file"`

	configMutex       sync.Mutex
	configFileWatcher config.FileWatcher
	schedules         *core.HeaterSchedules
}

// Check runs a single check
func (h *HeaterChecker) Check() {
	l := logging.NewLogger("Heater.Check").With().Str("module", h.GetName()).Logger()

	// Initial configuration and config change handling
	if h.schedules == nil {
		err := h.initSchedulesConfig()
		if err != nil {
			l.Error().Err(err).
				Str("filename", h.SchedulesFile).
				Msg("Unable to load configuration from file")
			return
		}

		// Temporize to let the FileWatcher load the configuration
		// Better to do that than a very complex sync system just for initialization (and risking deadlock issues...)
		time.Sleep(time.Second)
	}

	// Blocking if we are being reloading the configuration
	h.configMutex.Lock()
	defer h.configMutex.Unlock()

	if h.schedules == nil {
		l.Error().Err(errors.New("heater's schedules are not set")).Msg("Unable to Check heater")
		return
	}

	now := time.Now()

	// Getting climate entity
	climateEntity, err := httpclient.GetSimpleClient().GetEntity(h.schedules.Thermostat.Domain, h.schedules.Thermostat.EntityID)
	if err != nil {
		l.Error().Err(err).Msg("Unable to get current thermostat temperature")
		return
	}

	// Getting last seen entity for this climate
	if h.schedules.LastSeen.Enabled {
		lastSeenEntity, err := httpclient.GetSimpleClient().GetEntity(h.schedules.LastSeen.Entity.Domain, h.schedules.LastSeen.Entity.EntityID)
		if err != nil {
			l.Error().Err(err).
				Str("entity", h.schedules.LastSeen.Entity.GetEntityIDFullName()).
				Msg("Error getting last_seen entity, cannot set temperature")
			return
		}
		var lastSeenTime time.Time
		if h.schedules.LastSeen.ReadFromLastReported {
			lastSeenTime = lastSeenEntity.State.LastReported
		} else {
			lastSeenTime, err = time.Parse(time.RFC3339, lastSeenEntity.State.State)
			if err != nil {
				l.Warn().Err(err).
					Str("last_seen_state", lastSeenEntity.State.State).
					Str("entity", h.schedules.LastSeen.Entity.GetEntityIDFullName()).
					Msg("Unable to parse last_seen value")
				return
			}
		}
		if lastSeenTime.Add(h.schedules.LastSeen.OfflineAfter).Before(now) {
			l.Warn().
				Dur("duration", h.schedules.LastSeen.OfflineAfter).
				Str("entity", h.schedules.LastSeen.Entity.GetEntityIDFullName()).
				Msg("Entity has not been seen, setting it to off")
			if err := httpclient.GetSimpleClient().CallService(climateEntity, climateTurnOffService, map[string]interface{}{}); err != nil {
				l.Error().Err(err).
					Str("entity", climateEntity.GetEntityIDFullName()).
					Msg("Cannot turn off climate")
			}
			return
		}
		l.Info().
			Str("entity", h.schedules.LastSeen.Entity.GetEntityIDFullName()).
			Dur("duration", h.schedules.LastSeen.OfflineAfter).
			Time("last_seen", lastSeenTime).
			Msg("Entity has been seen soon enough, continuing setting temperature")
	}

	// Getting manual override status
	overrideEntity, err := httpclient.GetSimpleClient().GetEntity(h.schedules.ManualOverride.Domain, h.schedules.ManualOverride.EntityID)
	if err != nil {
		l.Warn().Err(err).Msg("Error getting manual_override entity from Home Assistant")
	}
	if overrideEntity.State.IsON() {
		l.Debug().Msg("manual_override is on, nothing to do")
		return
	}

	// Checking for dates first
	if h.schedules.DateBegin.After(now) && h.schedules.DateEnd.Before(now) {
		l.Debug().
			Time("current", now).
			Time("begin_date", time.Time(h.schedules.DateBegin)).
			Time("end_date", time.Time(h.schedules.DateEnd)).
			Msg("Current date is NOT between begin and end, nothing to do")
		// Ensuring heater climate is off
		if err := httpclient.GetSimpleClient().CallService(climateEntity, climateTurnOffService, map[string]interface{}{}); err != nil {
			l.Warn().Err(err).
				Str("entity", climateEntity.GetEntityIDFullName()).
				Msg("Cannot turn off climate")
		}
		return
	} else {
		l.Debug().
			Time("current", now).
			Time("begin_date", time.Time(h.schedules.DateBegin)).
			Time("end_date", time.Time(h.schedules.DateEnd)).
			Msg("Current date is between begin and end, configuring heaters following schedules")
	}

	// Ensuring climate is on
	if err := httpclient.GetSimpleClient().CallService(climateEntity, climateTurnOnService, map[string]interface{}{}); err != nil {
		l.Warn().Err(err).
			Str("entity", climateEntity.GetEntityIDFullName()).
			Msg("Cannot turn on climate, continuing anyway")
	}

	// Computing correct temperature depending on time
	tempToSet := h.schedules.GetTemperatureToSet(now)
	currentTemp, ok := (climateEntity.State.Attributes[temperatureAttributeName]).(float64)

	l = l.With().
		Str("climate", h.schedules.Thermostat.GetEntityIDFullName()).
		Float64("temp", tempToSet).
		Float64("cur_temp", currentTemp).Logger()

	if !ok || tempToSet != currentTemp {
		err := httpclient.GetSimpleClient().CallService(
			climateEntity,
			setTemperatureService,
			map[string]interface{}{
				temperatureAttributeName: tempToSet,
			})
		if err != nil {
			l.Error().Err(err).Msg("Unable to set new temperature for climate")
			return
		}

		l.Info().Msg("Setting new temperature for climate")
	} else {
		l.Debug().Msg("Temperature already set, nothing to do")
	}
}

func (h *HeaterChecker) initSchedulesConfig() error {
	if h.schedules != nil {
		return nil
	}

	l := logging.NewLogger("Heater.initSchedulesConfig")
	l.Info().Str("filename", h.SchedulesFile).Msg("Configuring heater schedules")

	h.configFileWatcher = config.NewFileWatcher(h.SchedulesFile, func() interface{} {
		return &core.HeaterSchedules{}
	})
	// Callback when reload is done, unlock the mutex to allow Check() to continue / to be called
	h.configFileWatcher.AddOnReloadCallbacks(func(data interface{}, err error) {
		if err == nil {
			h.configMutex.Lock()
			h.schedules = data.(*core.HeaterSchedules)
			defer h.configMutex.Unlock()
			h.printDebugSchedules()
		}
	})

	routines.AddRunnable(h.configFileWatcher)
	return h.configFileWatcher.Start()
}

func (h *HeaterChecker) printDebugSchedules() {
	l := logging.NewLogger("Heater.printDebugSchedules").With().Str("filename", h.SchedulesFile).Logger()
	l.Debug().EmbedObject(h.schedules).Msg("Reloading heater's config")
}

// GetDefaultEcoTemp returns the
func (h *HeaterChecker) GetDefaultEcoTemp() float64 {
	h.configMutex.Lock()
	defer h.configMutex.Unlock()
	if h.schedules == nil {
		return DefaultEcoTemp
	}
	return h.schedules.DefaultEco
}

var (
	errNoEntity = fmt.Errorf("no entity configured")
)

// GetClimateEntity returns the climate entity attached to the Heater's schedules object
func (h *HeaterChecker) GetClimateEntity() (model.HassEntity, error) {
	h.configMutex.Lock()
	defer h.configMutex.Unlock()
	if h.schedules == nil {
		return model.HassEntity{}, errNoEntity
	}
	return h.schedules.Thermostat, nil
}

// GetManualOverrideEntity returns the manual override entity attached to the Heater's schedules object
func (h *HeaterChecker) GetManualOverrideEntity() (model.HassEntity, error) {
	h.configMutex.Lock()
	defer h.configMutex.Unlock()
	if h.schedules == nil {
		return model.HassEntity{}, errNoEntity
	}
	return h.schedules.ManualOverride, nil
}

// GinHandler godoc
func (h *HeaterChecker) GinHandler(c *gin.Context) {
	obj := struct {
		*core.Module
		Name          string
		SchedulesFile string
		Schedules     *core.HeaterSchedules
	}{
		Module:        &h.Module,
		Name:          h.Name,
		SchedulesFile: h.SchedulesFile,
		Schedules:     h.schedules,
	}

	c.JSON(http.StatusOK, obj)
}
