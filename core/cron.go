package core

import (
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/nmaupu/gotomation/routines"
	"github.com/robfig/cron"
)

var (
	_ Configurable = (*CronEntry)(nil)
)

// Crontab is a Cron object
type Crontab interface {
	routines.Runnable
	AddFunc(spec string, cmd func()) error
}

type crontab struct {
	*cron.Cron
}

// NewCrontab returns a new pointer to a Crontab object
func NewCrontab() Crontab {
	return &crontab{
		Cron: cron.New(),
	}
}

func (c *crontab) Start() error {
	c.Cron.Start()
	return nil
}

func (c *crontab) Stop() {
	c.Cron.Stop()
}

func (c *crontab) AddFunc(spec string, cmd func()) error {
	return c.Cron.AddFunc(spec, cmd)
}

// GetName returns the name of this runnable object
func (c *crontab) GetName() string {
	return "Crontab"
}

// CronEntry is a struct to configure a crontab's entry
type CronEntry struct {
	Expr     string             `mapstructure:"expr"`
	Action   string             `mapstructure:"action"`
	Entities []model.HassEntity `mapstructure:"entities"`
}

// Configure reads the configuration and returns a new Checkable object
func (c *CronEntry) Configure(data interface{}, i interface{}) error {
	l := logging.NewLogger("CronEntry.Configure")

	err := config.NewMapStructureDecoder(c).Decode(data)
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
			httpclient.GetSimpleClient().CallService(entity, c.Action, nil)
		}
	}
}
