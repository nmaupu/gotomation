package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/module"
	"github.com/spf13/viper"
)

func main() {
	// Get config from file
	vi := viper.New()
	vi.SetConfigName("config")
	vi.SetConfigType("yaml")
	vi.AddConfigPath(".")
	vi.AddConfigPath("$HOME/.gotomation")
	vi.AddConfigPath("/etc/gotomation")
	vi.WatchConfig()
	vi.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Reloading configuration %s", e.Name)
		reloadConfig(vi)
	})

	// Load config when starting
	reloadConfig(vi)

	// Adding callbacks for server communication, start and subscribe to events
	httpclient.WebSocketClientSingleton.RegisterCallback("event", Event, model.HassEvent{})
	httpclient.WebSocketClientSingleton.Start()
	httpclient.WebSocketClientSingleton.SubscribeEvents("state_changed")
	httpclient.WebSocketClientSingleton.SubscribeEvents("roku_command")

	// Main loop, ctrl+c to stop
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-interrupt:
			log.Println("Stopping service")
			select {
			case <-time.After(time.Second):
				httpclient.WebSocketClientSingleton.Stop()
				module.StopAllModules()
			}
			return
		}
	}
}

// Event godoc
func Event(msg model.HassAPIObject) {
	//event := msg.(*model.HassEvent)
	//log.Printf("Received: event, msg=%+v\n", event)
}

func reloadConfig(vi *viper.Viper) {
	config := config.Gotomation{}

	if err := vi.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("No config file available")
		}

		log.Fatalf("Cannot read config file, err=%v", err)
	}

	if err := vi.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to unmarshal config file, err=%v", err)
	}

	// Init services and singletons
	httpclient.Init(config)
	module.Init(config)
}
