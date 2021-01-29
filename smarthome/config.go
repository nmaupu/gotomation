package smarthome

import (
	"sync"

	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/smarthome/checkers"
	"github.com/nmaupu/gotomation/smarthome/triggers"
	"github.com/robfig/cron"
)

var (
	// Checkers stores all checkers
	Checkers map[string]core.Checkable
	// Triggers stores all triggers
	Triggers map[string]core.Triggerable
	// mutex is used to lock map access by one goroutine only
	mutex sync.Mutex
	// cron
	crontab *cron.Cron
)

// Init inits modules from configuration
func Init(config config.Gotomation) {
	mutex.Lock()
	defer mutex.Unlock()

	initTriggers(&config)
	initCheckers(&config)
	initCrons(&config)
}

func initTriggers(config *config.Gotomation) {
	l := logging.NewLogger("initTriggers")
	Triggers = make(map[string]core.Triggerable, 0)

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

			Triggers[triggerName] = trigger
		}
	}
}

func initCheckers(config *config.Gotomation) {
	l := logging.NewLogger("initCheckers")

	// (Re)init checkers map
	Checkers = make(map[string]core.Checkable, 0)

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

			Checkers[moduleName] = checker
		}
	}

	StartAllCheckers()
}

// StopAllCheckers stops all checkers
func StopAllCheckers() {
	l := logging.NewLogger("StopAllCheckers")
	for name, checker := range Checkers {
		l.Info().
			Str("checker_name", name).
			Msg("Stopping checker")
		checker.Stop()
	}
}

// StartAllCheckers stops all checkers
func StartAllCheckers() {
	l := logging.NewLogger("StartAllCheckers")
	for name, checker := range Checkers {
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

// EventCallback is called when a listen event occurs
func EventCallback(msg model.HassAPIObject) {
	l := logging.NewLogger("EventCallback")
	mutex.Lock()
	defer mutex.Unlock()

	if Triggers == nil || len(Triggers) == 0 {
		return
	}

	event := msg.(*model.HassEvent)

	l.Trace().
		EmbedObject(event).
		Msg("Event received by the callback func")

	// Look for the entity
	for _, t := range Triggers {
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
