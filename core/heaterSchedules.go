package core

import (
	"time"

	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model/config"
)

var (
	_ Configurable = (*HeaterSchedules)(nil)
)

// HeaterSchedules stores all schedules for a heater
type HeaterSchedules struct {
	Scheds map[string][]HeaterSchedule `mapstructure:"schedules"`
}

// HeaterSchedule represents a heater's schedule
type HeaterSchedule struct {
	Beg     time.Time `mapstructure:"beg"`
	End     time.Time `mapstructure:"end"`
	Confort float64   `mapstructure:"confort"`
	Eco     float64   `mapstructure:"eco"`
}

// Configure reads the configuration and returns a new Checkable object
func (c *HeaterSchedules) Configure(data interface{}, i interface{}) error {
	l := logging.NewLogger("HeaterSchedules.Configure")

	err := config.NewMapStructureDecoder(c).Decode(data)
	if err != nil {
		return err
	}

	l.Trace().
		Msgf("%+v", c)

	return nil
}
