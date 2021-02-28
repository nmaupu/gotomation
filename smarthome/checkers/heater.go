package checkers

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/routines"
)

var (
	_ (core.Modular) = (*Heater)(nil)
)

// Heater sets the heater's thermostat based on schedules
type Heater struct {
	core.Module    `mapstructure:",squash"`
	Name           string           `mapstructure:"name"`
	SchedulesFile  string           `mapstructure:"schedules_file"`
	ManualOverride model.HassEntity `mapstructure:"manual_override"`
	Thermostat     model.HassEntity `mapstructure:"thermostat"`

	configMutex       sync.Mutex
	configFileWatcher config.FileWatcher
	schedules         *core.HeaterSchedules
}

// GetName godoc
func (h *Heater) GetName() string {
	return reflect.TypeOf(h).Elem().Name()
}

// Check runs a single check
func (h *Heater) Check() {
	l := logging.NewLogger("Heater.Check")

	// Initial configuration and config change handling
	if h.schedules == nil {
		err := h.initSchedulesConfig()
		if err != nil {
			l.Error().Err(err).
				Str("filename", h.SchedulesFile).
				Msg("Unable to load configuration from file")
			return
		}
	}

	// Blocking if we are being reloading the configuration
	h.configMutex.Lock()
	defer h.configMutex.Unlock()

	if h.schedules == nil {
		l.Error().Err(errors.New("Heater's schedules are not set"))
		return
	}

	// Getting manual override status
	overrideEntity, err := httpclient.GetSimpleClient().GetEntity(h.ManualOverride.Domain, h.ManualOverride.EntityID)
	if err != nil {
		l.Error().Err(err).Msg("Error getting manual_override entity from Home Assistant")
	}
	if overrideEntity.State.IsON() {
		l.Debug().Msg("manual_override is on, nothing to do")
		return
	}

	now := time.Now().Local()
	temp := h.schedules.GetTemperatureToSet(now)

	l.Info().
		Str("heater", h.Name).
		Str("schedules", fmt.Sprintf("%+v", h.schedules)).
		Float64("temperature", temp).
		Msg("Configuring heater's temperature")
}

func (h *Heater) initSchedulesConfig() error {
	if h.schedules != nil {
		return nil
	}

	l := logging.NewLogger("Heater.initSchedulesConfig")
	l.Info().Str("filename", h.SchedulesFile).Msg("Configuring heater schedules")

	h.configFileWatcher = config.NewFileWatcher(h.SchedulesFile, h.getSchedulesType)
	routines.AddRunnable(h.configFileWatcher)
	return h.configFileWatcher.Start()
}

func (h *Heater) getSchedulesType() interface{} {
	// Ensure locking access to configuration because if we are in Check when the watcher kicks in
	// we are going to be in trouble...
	h.configMutex.Lock()
	defer h.configMutex.Unlock()
	h.schedules = &core.HeaterSchedules{}
	return h.schedules
}
