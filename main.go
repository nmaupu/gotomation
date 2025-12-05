package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"

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

	// Loading sender configs from env if any and add them to the ones provided with the corresponding flag
	senderConfigsFromEnv := os.Getenv("SENDER_CONFIGS")
	if senderConfigsFromEnv != "" {
		gotoFlags.SenderConfig = append(gotoFlags.SenderConfig, strings.Split(senderConfigsFromEnv, `,`)...)
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

	// Get config from file
	vi := viper.New()
	vi.SetConfigType("yaml")
	vi.SetConfigName(filepath.Base(gotoFlags.ConfigFile))
	vi.AddConfigPath(filepath.Dir(gotoFlags.ConfigFile))
	vi.WatchConfig()

	// Binding some env var to config keys
	vi.BindEnv("home_assistant.token", "HASS_TOKEN")
	vi.BindEnv("open_mqtt_gateway.mqtt.username", "OMG_MQTT_USERNAME")
	vi.BindEnv("open_mqtt_gateway.mqtt.password", "OMG_MQTT_PASSWORD")
	vi.BindEnv("open_mqtt_gateway.mqtt.broker", "OMG_MQTT_BROKER")
	vi.BindEnv("open_mqtt_gateway.mqtt.prefix", "OMG_MQTT_PREFIX")

	vi.OnConfigChange(func(e fsnotify.Event) {
		l := logging.NewLogger("OnConfigChange")
		l.Info().Str("config", e.Name).Msg("Reloading configuration")

		_ = configChange(vi, gotoConfig, func(config config.Gotomation) error {
			smarthome.StopAndWait()
			return loadConfig(config)
		})
	})

	// Display binary information
	displayVersionInfo(l)

	// Load config when starting
	err := configChange(vi, gotoConfig, loadConfig)
	if err != nil {
		l.Fatal().Err(err).Msg("Unable to load config")
	}

	for _, s := range gotoConfig.Senders {
		l.Debug().Object("configured_sender", s).Send()
	}

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

func configChange(vi *viper.Viper, config config.Gotomation, loadFunc func(config config.Gotomation) error) error {
	l := logging.NewLogger("configChange")
	err := config.ReadConfigFromFile(vi, loadFunc)
	if err != nil {
		l.Error().Err(err).Msgf("An error occurred reloading configuration")
	}
	return err
}

func loadConfig(config config.Gotomation) error {
	return smarthome.Init(config)
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
	flag.StringVarP(&gotoFlags.HassToken, "token", "t", "", "Specify token to use for Home Assistant API calls, (env var HASS_TOKEN)")
	flag.StringSliceVar(&gotoFlags.SenderConfig, "senderConfig", []string{}, "Specify custom sender config as base64 json (multiple --configSender possible). This can be set by the env var SENDER_CONFIGS as a comma serparated string")

	flag.Parse()

	if gotoFlags.ConfigFile == "" {
		l.Fatal().Msg("Configuration file not provided")
	}

	if err := logging.SetVerbosity(gotoFlags.Verbosity); err != nil {
		_ = logging.SetVerbosity("info")
	}

	return gotoFlags
}
