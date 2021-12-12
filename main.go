package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/rs/zerolog"
	"math/rand"
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
	ConfigFile string
	Verbosity  string
	HassToken  string
	Version    bool

	SenderConfig []string
}

func main() {
	l := logging.NewLogger("main")
	gotoFlags := handleFlags()

	if gotoFlags.Version {
		displayVersionInfo(l)
		os.Exit(0)
	}

	gotoConfig := config.Gotomation{}
	if gotoFlags.HassToken != "" {
		gotoConfig.HomeAssistant.Token = gotoFlags.HassToken
	}

	for _, senderConfigJSONb64 := range gotoFlags.SenderConfig {
		senderCfg := config.SenderConfig{}
		senderConfigJSON, err := base64.StdEncoding.DecodeString(senderConfigJSONb64)
		if err != nil {
			l.Error().
				Err(err).
				Str("config", senderConfigJSONb64).
				Msg("An error occurred parsing --senderConfig")
			continue
		}

		err = json.Unmarshal(senderConfigJSON, &senderCfg)
		if err != nil {
			l.Error().
				Err(err).
				Str("config", string(senderConfigJSON)).
				Msg("An error occurred parsing --senderConfig")
			continue
		}
		gotoConfig.Senders = append(gotoConfig.Senders, senderCfg)
	}

	// Initializing rand package
	rand.Seed(time.Now().UnixNano())

	// Get config from file
	vi := viper.New()
	vi.SetConfigType("yaml")
	vi.SetConfigName(filepath.Base(gotoFlags.ConfigFile))
	vi.AddConfigPath(filepath.Dir(gotoFlags.ConfigFile))
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
	displayVersionInfo(l)

	// Main loop, ctrl+c or kill -15 to stop
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
	l := logging.NewLogger("configChange")
	err := config.ReadConfigFromFile(vi, loadFunc)
	if err != nil {
		l.Error().Err(err).Msgf("An error occurred reloading configuration")
	}
}

func loadConfig(config config.Gotomation) {
	smarthome.Init(config)
}

func displayVersionInfo(logger zerolog.Logger) {
	logger.Info().
		Str("version", app.ApplicationVersion).
		Str("build_date", app.BuildDate).
		Msg("Binary compilation info")
}

func handleFlags() gotomationFlags {
	l := logging.NewLogger("handleFlags")
	gotoFlags := gotomationFlags{}
	flag.BoolVar(&gotoFlags.Version, "version", false, "Display Version and exit")
	flag.StringVarP(&gotoFlags.ConfigFile, "config", "c", "gotomation.yaml", "Specify configuration file to use")
	flag.StringVarP(&gotoFlags.Verbosity, "verbosity", "v", "info", "Specify log's Verbosity")
	flag.StringVarP(&gotoFlags.HassToken, "token", "t", "", "Specify token to use for Home Assistant API calls")
	flag.StringSliceVar(&gotoFlags.SenderConfig, "senderConfig", []string{}, "Specify custom sender config as base64 json (multiple --configSender possible)")

	flag.Parse()

	if gotoFlags.ConfigFile == "" {
		l.Fatal().Msg("Configuration file not provided")
	}

	if err := logging.SetVerbosity(gotoFlags.Verbosity); err != nil {
		logging.SetVerbosity("info")
	}

	return gotoFlags
}
