package core

import (
	"github.com/mitchellh/mapstructure"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
)

var (
	_ Configurable = (*CronEntry)(nil)
)

// CronEntry is a struct to configure a cron entry
type CronEntry struct {
	Expr     string             `mapstructure:"expr"`
	Action   string             `mapstructure:"action"`
	Entities []model.HassEntity `mapstructure:"entities"`
}

// Configure reads the configuration and returns a new Checkable object
func (c *CronEntry) Configure(config interface{}, i interface{}) error {
	l := logging.NewLogger("CronEntry.Configure")

	mapstructureConfig := &mapstructure.DecoderConfig{
		DecodeHook: MapstructureDecodeHook,
		Result:     c,
	}
	decoder, _ := mapstructure.NewDecoder(mapstructureConfig)
	err := decoder.Decode(config)
	if err != nil {
		return err
	}

	l.Trace().
		Msgf("%+v", c)

	return nil
}

// GetActionFunc returns a func to execute when cron time is triggered
func (c *CronEntry) GetActionFunc() func() {
	return func() {
		for _, entity := range c.Entities {
			httpclient.SimpleClientSingleton.CallService(entity, c.Action)
		}
	}
}
