package checkers

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/routines"
)

const (
	temperatureAttributeName = "temperature"
	setTemperatureService    = "set_temperature"
)

var (
	_ (core.Modular) = (*Heater)(nil)
)

// Heater sets the heater's thermostat based on schedules
type Heater struct {
	core.Module   `mapstructure:",squash"`
	SchedulesFile string `mapstructure:"schedules_file"`

	configMutex       sync.Mutex
	configFileWatcher config.FileWatcher
	schedules         *core.HeaterSchedules
}

// Check runs a single check
func (h *Heater) Check() {
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
		l.Error().Err(errors.New("Heater's schedules are not set")).Msg("Unable to Check heater")
		return
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

	// Getting current temperature
	climateEntity, err := httpclient.GetSimpleClient().GetEntity(h.schedules.Thermostat.Domain, h.schedules.Thermostat.EntityID)
	if err != nil {
		l.Error().Err(err).Msg("Unable to get current thermostat temperature")
		return
	}

	// Computing correct temperature depending on time
	tempToSet := h.schedules.GetTemperatureToSet(time.Now())
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

func (h *Heater) initSchedulesConfig() error {
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

func (h *Heater) printDebugSchedules() {
	l := logging.NewLogger("Heater.printDebugSchedules").With().Str("filename", h.SchedulesFile).Logger()
	l.Debug().EmbedObject(h.schedules).Msg("Reloading heater's config")
}

// GinHandler godoc
func (h *Heater) GinHandler(c *gin.Context) {
	obj := struct {
		core.Module
		Name          string
		SchedulesFile string
		Schedules     core.HeaterSchedules
	}{
		Module:        h.Module,
		Name:          h.Name,
		SchedulesFile: h.SchedulesFile,
		Schedules:     *h.schedules,
	}

	c.JSON(http.StatusOK, obj)
}
