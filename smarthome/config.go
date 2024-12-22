package smarthome

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/nmaupu/gotomation/smarthome/messaging"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/httpservice"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/routines"
	"github.com/nmaupu/gotomation/thirdparty"
	"google.golang.org/api/calendar/v3"
)

const (
	// ModuleInternetChecker is a module to check the internet connection
	ModuleInternetChecker = "internetchecker"
	// ModuleHeaterChecker is a module to set heater temperature
	ModuleHeaterChecker = "heaterchecker"
	// ModuleCalendarChecker checks a calendar at regular interval
	ModuleCalendarChecker = "calendarchecker"
	// ModuleFreshness checks at a regular interval if device has been last seen not too long ago
	ModuleFreshness = "freshnesschecker"
	// ModuleTemperatureChecker checks at a regular interval if device exceeds a certain temperature and alert if it does
	ModuleTemperatureChecker = "temperaturechecker"
	// TriggerDehumidifier triggers dehumidifier on or off depending on humidity
	TriggerDehumidifier = "dehumidifier"
	// TriggerHarmony uses Roku Emulated to make actions based on Harmony remote buttons press
	TriggerHarmony = "harmony"
	// TriggerCalendarLights set to on or off lights based on calendar events
	TriggerCalendarLights = "calendarlights"
	// TriggerHeaterCheckersDisabler globally disables heaters' automatic programmation
	TriggerHeaterCheckersDisabler = "heatercheckersdisabler"
	// TriggerRandomLights is a module to set on/off lights randomly between a specific time frame
	TriggerRandomLights = "randomlights"
	// TriggerAlertBool is a module to send alerts to a specific sender depending on binary entity state change
	TriggerAlertBool = "alert"
)

var (
	checkers = map[string]func() core.Modular{
		ModuleInternetChecker: func() core.Modular {
			return new(InternetChecker)
		},
		ModuleHeaterChecker: func() core.Modular {
			return new(HeaterChecker)
		},
		ModuleCalendarChecker: func() core.Modular {
			return new(CalendarChecker)
		},
		ModuleFreshness: func() core.Modular {
			return new(FreshnessChecker)
		},
		ModuleTemperatureChecker: func() core.Modular {
			return new(TemperatureChecker)
		},
	}

	triggers = map[string]func() core.Actionable{
		TriggerDehumidifier: func() core.Actionable {
			return new(DehumidifierTrigger)
		},
		TriggerHarmony: func() core.Actionable {
			return new(HarmonyTrigger)
		},
		TriggerCalendarLights: func() core.Actionable {
			return new(CalendarLightsTrigger)
		},
		TriggerHeaterCheckersDisabler: func() core.Actionable {
			return new(HeaterCheckersDisablerTrigger)
		},
		TriggerRandomLights: func() core.Actionable {
			return new(RandomLightsTrigger)
		},
		TriggerAlertBool: func() core.Actionable {
			return new(AlertTriggerBool)
		},
	}
)

var (
	// mutex is used to lock maps' access by one goroutine only
	mutex     sync.RWMutex
	mCheckers map[string][]core.Checkable
	mTriggers map[string][]core.Triggerable
	crontab   core.Crontab
	mSenders  map[string]messaging.Sender
)

// Init inits checkers from configuration
func Init(config config.Gotomation) error {
	l := logging.NewLogger("Init")
	mutex.Lock()
	defer mutex.Unlock()

	routines.ResetRunnablesList()
	initHTTPClients(&config)

	if err := initZone(&config); err != nil {
		l.Error().Err(err).
			Str("zone_name", config.HomeAssistant.HomeZoneName).
			Msg("Unable to get coordinates from zone name")
		return err
	}

	initHTTPServer(&config)
	initGoogle(&config)
	initSenderConfigs(&config)
	initTriggers(&config)
	initCheckers(&config)
	initCrons(&config)
	routines.StartAllRunnables()
	return nil
}

// StopAndWait stops and free all allocated smarthome objects
func StopAndWait() {
	l := logging.NewLogger("Stop")

	l.Info().Msg("Stopping services")
	routines.StopAllRunnables()

	app.RoutinesWG.Wait()
	routines.ResetRunnablesList()
	l.Debug().Msg("All go routines terminated")
}

func initHTTPClients(config *config.Gotomation) {
	simpleClientScheme := "https"
	if !config.HomeAssistant.TLSEnabled {
		simpleClientScheme = "http"
	}
	httpclient.InitSimpleClient(simpleClientScheme, config.HomeAssistant.Host, config.HomeAssistant.Token, config.HomeAssistant.HealthCheckEntities)

	websocketClientScheme := "wss"
	if !config.HomeAssistant.TLSEnabled {
		websocketClientScheme = "ws"
	}
	httpclient.InitWebSocketClient(websocketClientScheme, config.HomeAssistant.Host, config.HomeAssistant.Token)
	routines.AddRunnable(httpclient.GetWebSocketClient())

	// Adding callbacks for server communication, start and subscribe to events
	httpclient.GetWebSocketClient().RegisterCallback("event", EventCallback, model.HassEvent{})
	httpclient.GetWebSocketClient().SubscribeEvents(config.HomeAssistant.SubscribeEvents...)
}

func initGoogle(config *config.Gotomation) {
	l := logging.NewLogger("initGoogle")

	if config.Google.CredentialsFile == "" {
		return
	}

	err := thirdparty.InitGoogleConfig(config.Google.CredentialsFile, calendar.CalendarReadonlyScope)
	if err != nil {
		l.Error().Err(err).Msg("Unable to init Google creds")
	}

	client, err := thirdparty.GetGoogleConfig().GetClient()
	if err != nil || client == nil {
		l.Error().Err(err).Msg("cannot get token from Google, allow Gotomation app first")
	}
}

func initSenderConfigs(config *config.Gotomation) {
	var err error
	l := logging.NewLogger("initSenderConfigs")

	// (Re)init sender configs map
	mSenders = make(map[string]messaging.Sender, 0)

	for _, senderConfig := range config.Senders {
		mSenders[senderConfig.Name], err = senderConfig.GetSender()
		if err != nil {
			l.Error().
				Err(err).
				Str("name", senderConfig.Name).
				Msg("Unable to configure sender")
		}
	}
}

func initTriggers(config *config.Gotomation) {
	l := logging.NewLogger("initTriggers")
	mTriggers = make(map[string][]core.Triggerable, 0)

	for _, trigger := range config.Triggers {
		for tn, triggerConfig := range trigger {
			// NOTE: due to an issue in Viper not being able to unmarshal map[string]any keys as case sensitive
			// we force it lower cased. Consequently, if the bug is one day fixed, this will continue to work as expected
			// See: https://github.com/spf13/viper/issues/1014
			triggerName := strings.ToLower(tn)
			trigger := new(core.Trigger)

			// Getting the action from the existing list
			ftrig, ok := triggers[triggerName]
			if !ok {
				l.Warn().
					Str("trigger", triggerName).
					Msg("Trigger not found")
				continue
			}
			action := ftrig()

			if err := trigger.Configure(triggerConfig, action); err != nil {
				l.Error().Err(err).
					Str("trigger", triggerName).
					Msg("Unable to decode configuration for trigger")
				continue
			}

			l.Info().
				Str("trigger", triggerName).
				Bool("enabled", trigger.Action.IsEnabled()).
				Msg("Initializing trigger")

			if mTriggers[triggerName] == nil {
				mTriggers[triggerName] = make([]core.Triggerable, 0)
			}
			mTriggers[triggerName] = append(mTriggers[triggerName], trigger)
		}
	}

	httpservice.HTTPServer().AddExtraHandlers(httpservice.GinConfigHandlers{
		Path:     "/trigger/:name",
		Handlers: []gin.HandlerFunc{triggerGinHandler},
	})

	// Call all triggers that needs an initialization with a dummy event
	for _, triggers := range mTriggers {
		for _, trig := range triggers {
			if trig.GetActionable().NeedsInitialization() {
				evt := model.DummyEvent // make a copy before passing a pointer
				trig.GetActionable().Trigger(&evt)
			}
		}
	}
}

func initCheckers(config *config.Gotomation) {
	l := logging.NewLogger("initCheckers")

	// (Re)init checkers map
	mCheckers = make(map[string][]core.Checkable, 0)

	for _, module := range config.Modules {
		for moduleName, moduleConfig := range module {
			checker := new(core.Checker)

			// Getting the module from the existing list
			fmod, ok := checkers[moduleName]
			if !ok {
				l.Warn().
					Str("module", moduleName).
					Msg("Module not found")
				continue
			}
			module := fmod()

			if err := checker.Configure(moduleConfig, module); err != nil {
				l.Error().Err(err).
					Str("module", moduleName).
					Msg("Unable to decode configuration for module")
				continue
			}

			l.Info().
				Str("module", moduleName).
				Bool("enabled", checker.Module.IsEnabled()).
				Msg("Initializing checker")

			if mCheckers[moduleName] == nil {
				mCheckers[moduleName] = make([]core.Checkable, 0)
			}
			mCheckers[moduleName] = append(mCheckers[moduleName], checker)
			routines.AddRunnable(checker)
		}
	}

	httpservice.HTTPServer().AddExtraHandlers(httpservice.GinConfigHandlers{
		Path:     "/checker/:name",
		Handlers: []gin.HandlerFunc{checkerGinHandler},
	})
}

func initCrons(config *config.Gotomation) {
	l := logging.NewLogger("initCrons")
	if crontab != nil {
		crontab.Stop()
	}

	crontab := core.NewCrontab()
	routines.AddRunnable(crontab)

	l.Info().Msg("Initializing all crons")
	for _, cronConfig := range config.Crons {
		ce := new(core.CronEntry)
		if err := ce.Configure(cronConfig, nil); err != nil {
			l.Error().Err(err).Msg("Unable to decode configuration for cron")
			continue
		}

		if err := crontab.AddFunc(ce.Expr, ce.GetActionFunc()); err != nil {
			l.Error().Err(err).
				Str("expr", ce.Expr).
				Msg("Unable to add func for cron")
		}
	}
}

func initZone(config *config.Gotomation) error {
	l := logging.NewLogger("initZone")

	var err error
	err = core.InitCoordinates(config.HomeAssistant.HomeZoneName)
	if err != nil {
		return err
	}
	routines.AddRunnable(core.Coords())

	l.Debug().
		Float64("latitude", core.Coords().GetLatitude()).
		Float64("longitude", core.Coords().GetLongitude()).
		Msg("GPS coordinates retrieved")

	return nil
}

func initHTTPServer(config *config.Gotomation) {
	//l := logging.NewLogger("initHTTPServer")

	httpservice.InitHTTPServer("0.0.0.0", httpservice.DefaultHTTPPort)
	routines.AddRunnable(httpservice.HTTPServer())
}

// EventCallback is called when a listen event occurs
func EventCallback(msg model.HassAPIObject) {
	l := logging.NewLogger("EventCallback")
	mutex.RLock()
	defer mutex.RUnlock()

	if mTriggers == nil || len(mTriggers) == 0 {
		return
	}

	event := msg.(*model.HassEvent)

	l.Trace().
		EmbedObject(event).
		Msg("Event received by the callback func")

	// Look for the entity
	for _, triggers := range mTriggers {
		for _, t := range triggers {
			if !t.GetActionable().IsEnabled() {
				continue
			}

			// Checking event types if defined
			toTriggerEvents := core.StringInSliceP(event.Event.EventType, t.GetActionable().GetEventTypesForTrigger())

			eventEntity := model.NewHassEntity(event.Event.Data.EntityID)
			toTriggerEntities := eventEntity.IsContained(t.GetActionable().GetEntitiesForTrigger())

			if toTriggerEvents || toTriggerEntities {
				// Call object's trigger func
				t.GetActionable().Trigger(event)
			}
		}
	}
}

func checkerGinHandler(c *gin.Context) {
	name := c.Params.ByName("name")

	for _, checkables := range mCheckers {
		for _, ch := range checkables {
			if path.Base(ch.GetName()) == name { // Removing any */ in the name
				ch.GetModular().GinHandler(c)
				return
			}
		}
	}

	c.AbortWithStatusJSON(http.StatusNotFound, model.NewAPIError(fmt.Errorf("Unable to find checker %s", name)))
}

func triggerGinHandler(c *gin.Context) {
	name := c.Params.ByName("name")

	for _, triggers := range mTriggers {
		for _, tr := range triggers {
			if path.Base(tr.GetName()) == name { // Removing any */ in the name
				tr.GetActionable().GinHandler(c)
				return
			}
		}
	}

	c.AbortWithStatusJSON(http.StatusNotFound, model.NewAPIError(fmt.Errorf("Unable to find trigger %s", name)))
}

// GetCheckersByType returns all checkers corresponding to a given name
func GetCheckersByType(name string) []core.Checkable {
	mutex.RLock()
	defer mutex.RUnlock()
	// As long as return value is a slice copy of interfaces,
	// we should be doing ok thread safe wise
	return mCheckers[name]
}

func GetSender(name string) messaging.Sender {
	mutex.RLock()
	defer mutex.RUnlock()
	sender, ok := mSenders[name]
	if !ok {
		return nil
	} else {
		return sender
	}
}
