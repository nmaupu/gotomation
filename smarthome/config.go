package smarthome

import (
	"sync"

	"github.com/nmaupu/gotomation/core"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
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
	Triggers = make(map[string]core.Triggerable, 0)

	for _, trigger := range config.Triggers {
		for triggerName, triggerConfig := range trigger {
			logging.Info("initTriggers").
				Str("trigger", triggerName).
				Msg("Initializing triggers")
			trigger := new(core.Trigger)
			var action core.Actionable

			switch triggerName {
			case "dehumidifier":
				action = new(DehumidifierTrigger)
			default:
				logging.Warn("config.initTriggers").
					Str("trigger", triggerName).
					Msg("Trigger not found")
				continue
			}

			if err := trigger.Configure(triggerConfig, action); err != nil {
				logging.Error("config.initTriggers").Err(err).
					Str("trigger", triggerName).
					Msg("Unable to decode configuration for trigger")
				continue
			}

			Triggers[triggerName] = trigger
		}
	}
}

// EventCallback is called when a listen event occurs
func EventCallback(msg model.HassAPIObject) {
	mutex.Lock()
	defer mutex.Unlock()

	if Triggers == nil || len(Triggers) == 0 {
		return
	}

	event := msg.(*model.HassEvent)

	// Look for the entity
	for _, t := range Triggers {
		if !t.GetActionable().IsEnabled() {
			continue
		}

		eventEntity := model.NewHassEntity(event.Event.Data.EntityID)
		if eventEntity.IsContained(t.GetActionable().GetEntitiesForTrigger()) {
			// Call object's trigger func
			t.GetActionable().Trigger(event)
		}
	}
}

func initCheckers(config *config.Gotomation) {
	if Checkers != nil && len(Checkers) > 0 {
		StopAllModules()
	}

	// (Re)init modules map
	Checkers = make(map[string]core.Checkable, 0)

	for _, module := range config.Modules {
		for moduleName, moduleConfig := range module {
			logging.Info("initCheckers").
				Str("module", moduleName).
				Msg("Initializing checkers")
			checker := new(core.Checker)
			var module core.Modular

			switch moduleName {
			case "internetChecker":
				module = new(InternetChecker)
			default:
				logging.Warn("config.initCheckers").
					Str("module", moduleName).
					Msg("Module not found")
				continue
			}

			if err := checker.Configure(moduleConfig, module); err != nil {
				logging.Error("config.initCheckers").Err(err).
					Str("module", moduleName).
					Msg("Unable to decode configuration for module")
				continue
			}

			Checkers[moduleName] = checker
		}
	}

	StartAllModules()
}

// StopAllModules stops all modules
func StopAllModules() {
	for name, module := range Checkers {
		logging.Info("config.StopAllModules").
			Str("module", name).
			Msg("Stopping module")
		module.Stop()
	}
}

// StartAllModules stops all modules
func StartAllModules() {
	for name, module := range Checkers {
		logging.Info("config.StartAllModules").
			Str("module", name).
			Msg("Starting module")
		err := module.Start()
		if err != nil {
			logging.Error("config.StartAllModules").Err(err).
				Str("module", name).
				Msg("Unable to start module")
		}
	}
}

func initCrons(config *config.Gotomation) {
	if crontab != nil {
		crontab.Stop()
	}

	crontab := cron.New()

	logging.Info("initCrons").Msg("Initializing all crons")
	for _, cronConfig := range config.Crons {
		ce := new(core.CronEntry)
		if err := ce.Configure(cronConfig, nil); err != nil {
			logging.Error("config.initCrons").Err(err).
				Msg("Unable to decode configuration for cron")
			continue
		}

		crontab.AddFunc(ce.Expr, ce.GetActionFunc())
	}

	crontab.Start()
}
