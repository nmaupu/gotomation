package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/smarthome"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type gotomationFlags struct {
	configFile string
	verbosity  string
	HassToken  string
}

func main() {
	l := logging.NewLogger("main")
	gotoFlags := handleFlags()

	gotoConfig := config.Gotomation{}
	if gotoFlags.HassToken != "" {
		gotoConfig.HomeAssistant.Token = gotoFlags.HassToken
	}

	// Get config from file
	vi := viper.New()
	vi.SetConfigType("yaml")
	vi.SetConfigName(filepath.Base(gotoFlags.configFile))
	vi.AddConfigPath(filepath.Dir(gotoFlags.configFile))
	vi.WatchConfig()
	vi.OnConfigChange(func(e fsnotify.Event) {
		l := logging.NewLogger("OnConfigChange")
		l.Info().Str("config", e.Name).Msg("Reloading configuration")

		configChange(vi, gotoConfig, func(config config.Gotomation) {
			smarthome.StopAndWait()
			loadConfig(config)
		})
	})

	// Load config when starting
	configChange(vi, gotoConfig, loadConfig)

	// Display binary information
	l.Info().
		Str("version", app.ApplicationVersion).
		Str("build_date", app.BuildDate).
		Msg("Binary compilation info")

	// Main loop, ctrl+c to stop
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-interrupt:
			smarthome.StopAndWait()
			return
		}
	}
}

func configChange(vi *viper.Viper, config config.Gotomation, loadFunc func(config config.Gotomation)) {
	l := logging.NewLogger("reloadConf")
	err := config.ReadConfigFromFile(vi, loadFunc)
	if err != nil {
		l.Error().Err(err).Msgf("An error occurred reloading configuration")
	}
}

func loadConfig(config config.Gotomation) {
	smarthome.Init(config)
}

func handleFlags() gotomationFlags {
	l := logging.NewLogger("handleFlags")
	gotoFlags := gotomationFlags{}
	flag.StringVarP(&gotoFlags.configFile, "config", "c", "gotomation.yaml", "Specify configuration file to use")
	flag.StringVarP(&gotoFlags.verbosity, "verbosity", "v", "info", "Specify log's verbosity")
	flag.StringVarP(&gotoFlags.HassToken, "token", "t", "", "Specify token to use for Home Assistant API calls")

	flag.Parse()

	if gotoFlags.configFile == "" {
		l.Fatal().Msg("Configuration file not provided")
	}

	if err := logging.SetVerbosity(gotoFlags.verbosity); err != nil {
		logging.SetVerbosity("info")
	}

	return gotoFlags
}
