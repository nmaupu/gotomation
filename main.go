package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/smarthome"
	"github.com/spf13/viper"
)

func main() {
	verbosity := flag.String("verbosity", "info", "Set log verbosity level")
	flag.Parse()
	err := logging.SetVerbosity(*verbosity)
	if err != nil {
		logging.Error("main").Err(err).Msg("Setting verbosity to default (info)")
		logging.SetVerbosity("info")
	}

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
	httpclient.WebSocketClientSingleton.RegisterCallback("event", smarthome.EventCallback, model.HassEvent{})
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
			logging.Info("main").Msg("Stopping service")
			select {
			case <-time.After(time.Second):
				httpclient.WebSocketClientSingleton.Stop()
				smarthome.StopAllModules()
			}
			return
		}
	}
}

func reloadConfig(vi *viper.Viper) {
	config := config.Gotomation{}

	if err := vi.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logging.Fatal("main.reloadConfig").Msg("No config file available")
		}

		logging.Fatal("main.reloadConfig").Err(err).Msg("Cannot read config file")
	}

	if err := vi.Unmarshal(&config); err != nil {
		logging.Fatal("main.reloadConfig").Err(err).Msg("Unable to unmarshal config file")
	}

	// Init services and singletons
	httpclient.Init(config)
	smarthome.Init(config)
}
