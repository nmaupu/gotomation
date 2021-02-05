package smarthome

import (
	"sync"

	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/httpservice"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/smarthome/checkers"
	"github.com/nmaupu/gotomation/smarthome/triggers"
	"github.com/nmaupu/gotomation/thirdparty"
	"github.com/robfig/cron"
	"google.golang.org/api/calendar/v3"
)

var (
	// mutex is used to lock map access by one goroutine only
	mutex      sync.Mutex
	mCheckers  map[string]core.Checkable
	mTriggers  map[string]core.Triggerable
	crontab    *cron.Cron
	httpServer *httpservice.HTTPService
)

// Init inits modules from configuration
func Init(config config.Gotomation) {
	l := logging.NewLogger("Init")
	mutex.Lock()
	defer mutex.Unlock()

	if err := initZone(&config); err != nil {
		l.Error().Err(err).
			Str("zone_name", config.HomeAssistant.HomeZoneName).
			Msg("Unable to get coordinate from zone name")
		return
	}

	initGoogle(&config)
	initTriggers(&config)
	initCheckers(&config)
	initCrons(&config)
	initHTTPServer(&config)
}

// StopAndWait stops and free all allocated smarthome object
func StopAndWait() {
	l := logging.NewLogger("Stop")

	l.Info().Msg("Stopping service")
	StopSunriseSunset()
	StopAllCheckers()
	StopCron()

	if httpclient.WebSocketClientSingleton != nil {
		httpclient.WebSocketClientSingleton.Stop()
	}

	StopHTTPServer()

	app.RoutinesWG.Wait()
	l.Debug().Msg("All go routines terminated")
}

func initGoogle(config *config.Gotomation) {
	l := logging.NewLogger("initGoogle")

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
	mTriggers = make(map[string]core.Triggerable, 0)

	for _, trigger := range config.Triggers {
		for triggerName, triggerConfig := range trigger {
			trigger := new(core.Trigger)
			var action core.Actionable

			switch triggerName {
			case "dehumidifier":
				action = new(triggers.Dehumidifier)
			case "harmony":
				action = new(triggers.Harmony)
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

			mTriggers[triggerName] = trigger
		}
	}
}

func initCheckers(config *config.Gotomation) {
	l := logging.NewLogger("initCheckers")

	// (Re)init checkers map
	mCheckers = make(map[string]core.Checkable, 0)

	for _, module := range config.Modules {
		for moduleName, moduleConfig := range module {
			checker := new(core.Checker)
			var module core.Modular

			switch moduleName {
			case "internetChecker":
				module = new(checkers.Internet)
			default:
				l.Warn().
					Str("module", moduleName).
					Msg("Module not found")
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

			mCheckers[moduleName] = checker
		}
	}

	StartAllCheckers()
}

// StopAllCheckers stops all checkers
func StopAllCheckers() {
	l := logging.NewLogger("StopAllCheckers")
	for name, checker := range mCheckers {
		l.Info().
			Str("checker_name", name).
			Msg("Stopping checker")
		checker.Stop()
	}
}

// StartAllCheckers stops all checkers
func StartAllCheckers() {
	l := logging.NewLogger("StartAllCheckers")
	for name, checker := range mCheckers {
		l.Info().
			Str("checker_name", name).
			Msg("Starting checker")
		err := checker.Start()
		if err != nil {
			l.Error().Err(err).
				Str("checker_name", name).
				Msg("Unable to start checker")
		}
	}
}

func initCrons(config *config.Gotomation) {
	l := logging.NewLogger("initCrons")
	if crontab != nil {
		crontab.Stop()
	}

	crontab := cron.New()

	l.Info().Msg("Initializing all crons")
	for _, cronConfig := range config.Crons {
		ce := new(core.CronEntry)
		if err := ce.Configure(cronConfig, nil); err != nil {
			l.Error().Err(err).Msg("Unable to decode configuration for cron")
			continue
		}

		crontab.AddFunc(ce.Expr, ce.GetActionFunc())
	}

	crontab.Start()
}

// StopCron stops cron and free associated resources
func StopCron() {
	if crontab != nil {
		crontab.Stop()
	}
}

// StopSunriseSunset stops the sunrise/sunset refresh goroutine
func StopSunriseSunset() {
	core.Coords().Stop()
}

func initZone(config *config.Gotomation) error {
	l := logging.NewLogger("initZone")

	var err error
	err = core.InitCoordinates(config.HomeAssistant.HomeZoneName)
	if err != nil {
		return err
	}

	l.Debug().
		Float64("latitude", core.Coords().GetLatitude()).
		Float64("longitude", core.Coords().GetLongitude()).
		Msg("GPS coordinates retrieved")

	l.Debug().Msg("Preloading sunrise and sunset dates (long to calculate on PI)")

	core.Coords().Start()

	return nil
}

func initHTTPServer(config *config.Gotomation) {
	//l := logging.NewLogger("initHTTPServer")

	httpservice.InitHTTPServer("127.0.0.1", httpservice.DefaultHTTPPort)
	httpservice.HTTPServer().Start()
}

// StopHTTPServer stops the HTTP server
func StopHTTPServer() {
	httpservice.HTTPServer().Stop()
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
	for _, t := range mTriggers {
		if !t.GetActionable().IsEnabled() {
			continue
		}

		// Checking event types if defined
		toTriggerEvents := core.StringInSlice(event.Event.EventType, t.GetActionable().GetEventTypesForTrigger())

		eventEntity := model.NewHassEntity(event.Event.Data.EntityID)
		toTriggerEntities := eventEntity.IsContained(t.GetActionable().GetEntitiesForTrigger())

		if toTriggerEvents || toTriggerEntities {
			// Call object's trigger func
			t.GetActionable().Trigger(event)
		}
	}
}
