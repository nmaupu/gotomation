package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nmaupu/gotomation/app"
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
		l := logging.NewLogger("OnConfigChange")
		l.Info().Str("config", e.Name).Msg("Reloading configuration")
		loadConfig(vi, true)
	})

	// Load config when starting
	loadConfig(vi, false)

	// Display binary information
	l.Info().
		Str("version", app.ApplicationVersion).
		Str("build_date", app.BuildDate).
		Msg("Binary compilation info")

	// Main loop, ctrl+c to stop
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-interrupt:
			stop()
			return
		}
	}
}

func handleFlags() gotomationFlags {
	l := logging.NewLogger("handleFlags")
	gotoFlags := gotomationFlags{}
	flag.StringVarP(&gotoFlags.configFile, "config", "c", "gotomation.yaml", "Specify configuration file to use")
	flag.StringVarP(&gotoFlags.verbosity, "verbosity", "v", "info", "Specify log's verbosity")

	flag.Parse()

	if gotoFlags.configFile == "" {
		l.Fatal().Msg("Configuration file not provided")
	}

	setLogLevel(gotoFlags.verbosity)

	return gotoFlags
}

func setLogLevel(lvl string) {
	l := logging.NewLogger("setLogLevel")

	err := logging.SetVerbosity(lvl)
	if err != nil {
		l.Error().Err(err).Msg("Setting verbosity to default (info)")
		logging.SetVerbosity("info")
	}
}

func stop() {
	l := logging.NewLogger("stop")

	l.Info().Msg("Stopping service")
	smarthome.StopAllCheckers()
	smarthome.StopCron()
	if httpclient.WebSocketClientSingleton != nil {
		httpclient.WebSocketClientSingleton.Stop()
	}

	app.RoutinesWG.Wait()
	l.Debug().Msg("All go routines terminated")
}

func loadConfig(vi *viper.Viper, isReloading bool) {
	l := logging.NewLogger("reloadConfig").With().Str("config_file", vi.ConfigFileUsed()).Logger()
	config := config.Gotomation{}
	reloadSleepDur := 5 * time.Second

	if err := vi.ReadInConfig(); err != nil {
		l.Error().Err(err).Msgf("Unable to read config file, retrying in %s", reloadSleepDur.String())
		time.Sleep(reloadSleepDur)
		loadConfig(vi, isReloading)
		return
	}

	if err := vi.Unmarshal(&config); err != nil {
		l.Error().Err(err).Msg("Unable to unmarshal config file")
		loadConfig(vi, isReloading)
		return
	}

	if !config.Validate() { // On some systems (rpi), reload succeeds but returns an empty object...
		l.Error().Err(fmt.Errorf("Config is not valid or is empty, retrying to reload in %s", reloadSleepDur.String())).Send()
		time.Sleep(reloadSleepDur)
		loadConfig(vi, isReloading)
		return
	}

	if config.LogLevel != "" {
		l.Info().Str("log_level", config.LogLevel).Msg("Setting log level using configuration file's value")
		setLogLevel(config.LogLevel)
	}
	l.Trace().Str("config", fmt.Sprintf("%+v", config)).Msg("Config dump")

	// Stopping only when reloading
	if isReloading {
		stop()
	}
	httpclient.Init(config)
	smarthome.Init(config)

	// Adding callbacks for server communication, start and subscribe to events
	httpclient.WebSocketClientSingleton.RegisterCallback("event", smarthome.EventCallback, model.HassEvent{})
	httpclient.WebSocketClientSingleton.Start()
	for _, sub := range config.HomeAssistant.SubscribeEvents {
		httpclient.WebSocketClientSingleton.SubscribeEvents(sub)
	}

}
