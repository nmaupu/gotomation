package smarthome

import (
	"fmt"
	"net/http"
	"path"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/httpservice"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/routines"
	"github.com/nmaupu/gotomation/smarthome/checkers"
	"github.com/nmaupu/gotomation/smarthome/triggers"
	"github.com/nmaupu/gotomation/thirdparty"
	"google.golang.org/api/calendar/v3"
)

var (
	// ModuleInternetChecker is a module to check the internet connection
	ModuleInternetChecker = "internetChecker"
	// ModuleHeaterChecker is a module to set heater temperature
	ModuleHeaterChecker = "heaterChecker"
	// ModuleCalendarChecker checks a calendar at regular interval
	ModuleCalendarChecker = "calendarChecker"
	// TriggerDehumidifier triggers dehumidifier on or off depending on humidity
	TriggerDehumidifier = "dehumidifier"
	// TriggerHarmony uses Roku Emulated to make actions based on Harmony remote buttons press
	TriggerHarmony = "harmony"
	// TriggerCalendarLights set to on or off lights based on calendar events
	TriggerCalendarLights = "calendarLights"
)

var (
	// mutex is used to lock maps' access by one goroutine only
	mutex     sync.Mutex
	mCheckers map[string][]core.Checkable
	mTriggers map[string][]core.Triggerable
	crontab   core.Crontab
)

// Init inits modules from configuration
func Init(config config.Gotomation) {
	l := logging.NewLogger("Init")
	mutex.Lock()
	defer mutex.Unlock()

	routines.ResetRunnablesList()
	initHTTPClients(&config)

	if err := initZone(&config); err != nil {
		l.Error().Err(err).
			Str("zone_name", config.HomeAssistant.HomeZoneName).
			Msg("Unable to get coordinates from zone name")
		return
	}

	initHTTPServer(&config)
	initGoogle(&config)
	initTriggers(&config)
	initCheckers(&config)
	initCrons(&config)
	routines.StartAllRunnables()
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
	httpclient.InitSimpleClient(config.HomeAssistant.Host, config.HomeAssistant.Token)

	httpclient.InitWebSocketClient(config.HomeAssistant.Host, config.HomeAssistant.Token)
	routines.AddRunnable(httpclient.GetWebSocketClient())

	// Adding callbacks for server communication, start and subscribe to events
	httpclient.GetWebSocketClient().RegisterCallback("event", EventCallback, model.HassEvent{})
	for _, sub := range config.HomeAssistant.SubscribeEvents {
		httpclient.GetWebSocketClient().SubscribeEvents(sub)
	}
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
		l.Error().Err(err).Msg("Cannot get token from Google, allow Gotomation app first")
	}
}

func initTriggers(config *config.Gotomation) {
	l := logging.NewLogger("initTriggers")
	mTriggers = make(map[string][]core.Triggerable, 0)

	for _, trigger := range config.Triggers {
		for triggerName, triggerConfig := range trigger {
			trigger := new(core.Trigger)
			var action core.Actionable

			switch triggerName {
			case TriggerDehumidifier:
				action = new(triggers.Dehumidifier)
			case TriggerHarmony:
				action = new(triggers.Harmony)
			case TriggerCalendarLights:
				action = new(triggers.CalendarLights)
			default:
				l.Warn().
					Str("trigger", triggerName).
					Msg("Trigger not found")
				continue
			}

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
}

func initCheckers(config *config.Gotomation) {
	l := logging.NewLogger("initCheckers")

	// (Re)init checkers map
	mCheckers = make(map[string][]core.Checkable, 0)

	for _, module := range config.Modules {
		for moduleName, moduleConfig := range module {
			checker := new(core.Checker)
			var module core.Modular

			switch moduleName {
			case ModuleInternetChecker:
				module = new(checkers.Internet)
			case ModuleCalendarChecker:
				module = new(checkers.Calendar)
			case ModuleHeaterChecker:
				module = new(checkers.Heater)
			default:
				l.Error().Err(fmt.Errorf("Cannot find module")).
					Str("module", moduleName).
					Send()
				continue
			}

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

		crontab.AddFunc(ce.Expr, ce.GetActionFunc())
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

	httpservice.InitHTTPServer("127.0.0.1", httpservice.DefaultHTTPPort)
	routines.AddRunnable(httpservice.HTTPServer())
}

// EventCallback is called when a listen event occurs
func EventCallback(msg model.HassAPIObject) {
	l := logging.NewLogger("EventCallback")
	mutex.Lock()
	defer mutex.Unlock()

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
