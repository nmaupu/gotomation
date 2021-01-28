package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/smarthome"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type gotomationFlags struct {
	configFile string
	verbosity  string
}

func main() {
	l := logging.NewLogger("main")
	gotoFlags := handleFlags()

	// Get config from file
	vi := viper.New()
	vi.SetConfigType("yaml")
	vi.SetConfigName(filepath.Base(gotoFlags.configFile))
	vi.AddConfigPath(filepath.Dir(gotoFlags.configFile))
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
			l.Info().Msg("Stopping service")
			select {
			case <-time.After(time.Second):
				httpclient.WebSocketClientSingleton.Stop()
				smarthome.StopAllModules()
			}
			return
		}
	}
}

func handleFlags() gotomationFlags {
	l := logging.NewLogger("handleFlags")
	gotoFlags := gotomationFlags{}
	flag.StringVarP(&gotoFlags.configFile, "config", "c", "gotomation.yaml", "Specify configuration file to use")
	flag.StringVarP(&gotoFlags.verbosity, "verbosity", "v", "info", "Set log verbosity level")

	flag.Parse()

	if gotoFlags.configFile == "" {
		l.Fatal().Msg("Configuration file not provided")
	}

	err := logging.SetVerbosity(gotoFlags.verbosity)
	if err != nil {
		l.Error().Err(err).Msg("Setting verbosity to default (info)")
		logging.SetVerbosity("info")
	}

	return gotoFlags
}

func reloadConfig(vi *viper.Viper) {
	l := logging.NewLogger("reloadConfig").With().Str("config_file", vi.ConfigFileUsed()).Logger()
	config := config.Gotomation{}

	if err := vi.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			l.Fatal().Msg("Unable to read config file")
		}

		l.Fatal().Err(err).Msg("Cannot read config file")
	}

	if err := vi.Unmarshal(&config); err != nil {
		l.Fatal().Err(err).Msg("Unable to unmarshal config file")
	}

	// Init services and singletons
	httpclient.Init(config)
	smarthome.Init(config)
}
