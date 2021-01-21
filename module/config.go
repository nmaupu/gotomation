package module

import (
	"log"

	"github.com/nmaupu/gotomation/model/config"
)

var (
	// Checkers stores all checkers
	Checkers map[string]Checkable
)

// Init inits modules from configuration
func Init(config config.Gotomation) {
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
