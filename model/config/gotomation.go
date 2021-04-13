package config

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	// TimeLayout used for time configuration
	TimeLayout = "15:04:05"
)

// Gotomation is the struct to unmarshal configuration
// It is using mapstructure for a compatibility with Viper config files
type Gotomation struct {
	// LogLevel is the log level configured
	LogLevel string `mapstructure:"log_level"`
	// Google is used to authenticate to Google's API
	Google struct {
		CredentialsFile string `mapstructure:"creds_file"`
	} `mapstructure:"google"`
	// HomeAssistant server related options
	HomeAssistant struct {
		Host                string             `mapstructure:"host"`
		Token               string             `mapstructure:"token"`
		SubscribeEvents     []string           `mapstructure:"subscribe_events"`
		HomeZoneName        string             `mapstructure:"home_zone_name"`
		TLSEnabled          bool               `mapstructure:"tls_enabled"`
		HealthCheckEntities []model.HassEntity `mapstructure:"health_check_entities"`
	} `mapstructure:"home_assistant"`

	// Modules configuration
	Modules []map[string]interface{} `mapstructure:"modules"`

	// Triggers configuration
	Triggers []map[string]interface{} `mapstructure:"triggers"`

	// Crons configuration
	Crons []interface{} `mapstructure:"crons"`
}

// Validate indicates whether or not the config is valid for gotomation to run
func (g Gotomation) Validate() bool {
	return g.HomeAssistant.Host != "" && g.HomeAssistant.Token != ""
}

// ReadConfigFromFile loads or reloads config from viper's config file
func (g Gotomation) ReadConfigFromFile(vi *viper.Viper, loadConfig func(config Gotomation)) error {
	l := logging.NewLogger("Gotomation.LoadConfig").With().Str("config_file", vi.ConfigFileUsed()).Logger()

	if err := vi.ReadInConfig(); err != nil {
		return errors.Wrap(err, "unable to read config file")
	}

	if err := vi.Unmarshal(&g); err != nil {
		return errors.Wrap(err, "unable to unmarshal config file")
	}

	if !g.Validate() { // On some systems (rpi), reload succeeds but returns an empty object for obscure reasons...
		return fmt.Errorf("config is not valid: Home Assistant host and token must be specified")
	}

	if g.LogLevel != "" {
		l.Info().Str("log_level", g.LogLevel).Msg("Setting log level using configuration file's value")
		err := logging.SetVerbosity(g.LogLevel)
		if err != nil {
			l.Error().Err(err).Msg("Setting verbosity to default (info)")
			logging.SetVerbosity("info")
		}
	}
	l.Trace().Str("config", fmt.Sprintf("%+v", g)).Msg("Config dump")

	loadConfig(g)
	return nil
}

// MapstructureDecodeHookFunc returns a mapstructure decode hook func to handle Gotomation configuration objects
func MapstructureDecodeHookFunc() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToTimeHookFunc(TimeLayout),
		model.StringToHassEntityDecodeHookFunc(),
		model.StringToDayMonthDateDecodeHookFunc(),
	)
}

// NewMapstructureDecoder returns a new mapstructure.Decoder ready to read Gotomation configuration
func NewMapstructureDecoder(result interface{}) *mapstructure.Decoder {
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook: MapstructureDecodeHookFunc(),
		Result:     result,
	}
	decoder, _ := mapstructure.NewDecoder(decoderConfig)
	return decoder
}
