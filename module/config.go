package module

import (
	"log"

	"github.com/nmaupu/gotomation/model/config"
)

var (
	// Modules stores all checker modules
	Modules map[string]Checkable
)

// Init inits modules from configuration
func Init(config config.Gotomation) {
	if Modules != nil && len(Modules) > 0 {
		StopAllModules()
	}

	// (Re)init modules map
	Modules = make(map[string]Checkable, 0)

	for _, module := range config.Modules {
		for moduleName, moduleConfig := range module {

			var module Checkable

			switch moduleName {
			case "freeboxChecker":
				module = new(FreeboxChecker)
			default:
				log.Printf("Module %s not found", moduleName)
				continue
			}

			if err := module.Configure(moduleConfig); err != nil {
				log.Printf("Unable to decode configuration for module %s, err=%v", moduleName, err)
				continue
			}

			Modules[moduleName] = module
		}
	}

	StartAllModules()
}

// StopAllModules stops all modules
func StopAllModules() {
	for name, module := range Modules {
		log.Printf("Stopping module %s", name)
		module.Stop()
	}
}

// StartAllModules stops all modules
func StartAllModules() {
	for name, module := range Modules {
		log.Printf("Starting module %s", name)
		module.Start()
	}
}
