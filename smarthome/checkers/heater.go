package checkers

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
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/routines"
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
	overrideEntity, err := httpclient.GetSimpleClient().GetEntity(h.schedules.ManualOverride.Domain, h.schedules.ManualOverride.EntityID)
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
	h.configFileWatcher.AddOnReloadCallbacks(func(data interface{}) {
		h.printDebugSchedules()
	})
	routines.AddRunnable(h.configFileWatcher)
	return h.configFileWatcher.Start()
}

func (h *Heater) printDebugSchedules() {
	l := logging.NewLogger("Heater.printDebugSchedules").With().Str("filename", h.SchedulesFile).Logger()
	l.Debug().EmbedObject(h.schedules).Msg("Reloading heater's config")
}

func (h *Heater) getSchedulesType() interface{} {
	// Ensure locking access to configuration because if we are in Check when the watcher kicks in
	// we are going to be in trouble...
	h.configMutex.Lock()
	defer h.configMutex.Unlock()
	h.schedules = &core.HeaterSchedules{}
	return h.schedules
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
