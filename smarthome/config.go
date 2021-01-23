package smarthome

import (
	"log"
	"sync"

	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/robfig/cron"
)

var (
	// Checkers stores all checkers
	Checkers map[string]Checkable
	// Triggers stores all triggers
	Triggers map[string]Triggerable
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
	Triggers = make(map[string]Triggerable, 0)

	for _, trigger := range config.Triggers {
		for triggerName, triggerConfig := range trigger {
			trigger := new(Trigger)
			var action Actionable

			switch triggerName {
			case "dehumidifier":
				action = new(DehumidifierTrigger)
			default:
				log.Printf("Trigger %s not found", triggerName)
				continue
			}

			if err := trigger.Configure(triggerConfig, action); err != nil {
				log.Printf("Unable to decode configuration for trigger %s, err=%v", triggerName, err)
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
	Checkers = make(map[string]Checkable, 0)

	for _, module := range config.Modules {
		for moduleName, moduleConfig := range module {
			checker := new(Checker)
			var module Modular

			switch moduleName {
			case "internetChecker":
				module = new(InternetChecker)
			default:
				log.Printf("Module %s not found", moduleName)
				continue
			}

			if err := checker.Configure(moduleConfig, module); err != nil {
				log.Printf("Unable to decode configuration for module %s, err=%v", moduleName, err)
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
		log.Printf("Stopping module %s", name)
		module.Stop()
	}
}

// StartAllModules stops all modules
func StartAllModules() {
	for name, module := range Checkers {
		log.Printf("Starting module %s", name)
		err := module.Start()
		if err != nil {
			log.Printf("Unable to start %s, err=%v", name, err)
		}
	}
}

func initCrons(config *config.Gotomation) {
	if crontab != nil {
		crontab.Stop()
	}

	crontab := cron.New()

	for _, cronConfig := range config.Crons {
		ce := new(CronEntry)
		if err := ce.Configure(cronConfig, nil); err != nil {
			log.Printf("Unable to decode configuration for cron, err=%v", err)
			continue
		}

		crontab.AddFunc(ce.Expr, ce.GetActionFunc())
	}

	crontab.Start()
}
