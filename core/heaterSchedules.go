package core

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model"
	"github.com/nmaupu/gotomation/model/config"
	"github.com/rs/zerolog"
)

const (
	Sunday = 1 << iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
	Week
	WeekEnd
)

var (
	_ Configurable = (*HeaterSchedules)(nil)

	days = []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "week", "weekend"}
)

// SchedulesDays are the days when the schedules applies
// Examples:
//  - week
//  - monday,tuesday,friday
//  - monday,wednesday,weekend
type SchedulesDays string

// HeaterSchedules stores all schedules for a heater
type HeaterSchedules struct {
	Scheds         map[SchedulesDays][]HeaterSchedule `mapstructure:"schedules"`
	DefaultEco     float64                            `mapstructure:"default_eco"`
	ManualOverride model.HassEntity                   `mapstructure:"manual_override"`
	Thermostat     model.HassEntity                   `mapstructure:"thermostat"`
	DateBegin      model.DayMonthDate                 `mapstructure:"date_begin"`
	DateEnd        model.DayMonthDate                 `mapstructure:"date_end"`
}

// HeaterSchedule represents a heater's schedule
type HeaterSchedule struct {
	Beg     time.Time `mapstructure:"beg"`
	End     time.Time `mapstructure:"end"`
	Comfort float64   `mapstructure:"comfort"`
	Eco     float64   `mapstructure:"eco"`
}

func getTodayTime(now time.Time, t time.Time, loc *time.Location) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

// TodayBeg returns c.Beg time with today's date
func (c HeaterSchedule) TodayBeg(now time.Time, loc *time.Location) time.Time {
	return getTodayTime(now, c.Beg, loc)
}

// TodayEnd returns c.End time with today's date
func (c HeaterSchedule) TodayEnd(now time.Time, loc *time.Location) time.Time {
	return getTodayTime(now, c.End, loc)
}

// IsActive returns true if given 't' is between c.Beg and c.End
func (c HeaterSchedule) IsActive(t time.Time) bool {
	l := logging.NewLogger("HeaterSchedule.IsActive")
	ret := t.After(c.TodayBeg(t, t.Location())) && t.Before(c.TodayEnd(t, t.Location()))
	l.Debug().
		EmbedObject(c).
		Bool("ret", ret).
		Time("t", t).Msg("Checking if schedule is active")
	return ret
}

// MarshalZerologObject godoc
func (c HeaterSchedule) MarshalZerologObject(event *zerolog.Event) {
	event.
		Time("beg", c.Beg).
		Time("end", c.End).
		Float64("confort", c.Comfort).
		Float64("eco", c.Eco)
}

// AsFlag returns an int from a SchedulesDays
func (s SchedulesDays) AsFlag() int {
	result := 0
	strs := strings.Split(string(s), ",")
	for _, str := range strs {
		idx := getSliceIdx(str, days)
		if idx < 0 {
			continue
		}

		if idx >= 0 && idx <= 6 {
			result |= 1 << idx
		} else if 1<<idx == Week {
			result |= Monday | Tuesday | Wednesday | Thursday | Friday
		} else if 1<<idx == WeekEnd {
			result |= Saturday | Sunday
		}
	}
	return result
}

func getSliceIdx(s string, sl []string) int {
	res := -1
	for k, v := range sl {
		if strings.EqualFold(strings.Trim(s, " "), v) {
			return k
		}
	}
	return res
}

// IsScheduled returns true if the day of 't' is contained into 's' SchedulesDays
func (s SchedulesDays) IsScheduled(t time.Time) bool {
	flag := s.AsFlag()
	currentDayFlag := 1 << t.Weekday()
	return currentDayFlag&flag == currentDayFlag
}

// Sort sorts schedules
func (c *HeaterSchedules) Sort() {
	for k, v := range c.Scheds {
		scheds := v
		sort.Slice(scheds, func(i, j int) bool {
			// Returns true if i elt > j elt
			return scheds[i].End.Before(scheds[j].Beg)
		})

		c.Scheds[k] = scheds
	}
}

// GetTemperatureToSet returns the temperature to set corresponding to the time given in parameter
func (c *HeaterSchedules) GetTemperatureToSet(t time.Time) float64 {
	if t.Location() == nil {
		t = t.Local()
	}

	finalTemp := c.DefaultEco
	// Sorting schedules to get stuff in order (eco temp of the previous time range
	// is the temperature to set if we are not currently in a "confort" time range)
	c.Sort()
	for schedulesDays, schedules := range c.Scheds {
		if !schedulesDays.IsScheduled(t) { // not schedules for today
			continue
		}

		// Configuration applies today
		for _, sched := range schedules {
			if sched.IsActive(t) { // in between
				return sched.Comfort
			}

			if t.After(sched.TodayEnd(t, t.Location())) {
				finalTemp = sched.Eco
			}
		}
	}

	return finalTemp
}

// MarshalZerologObject godoc
func (c *HeaterSchedules) MarshalZerologObject(event *zerolog.Event) {
	event.Time("date_begin", time.Time(c.DateBegin))
	event.Time("date_end", time.Time(c.DateEnd))
	for schedName, s := range c.Scheds {
		for idx, sched := range s {
			event = event.Object(fmt.Sprintf("%s[%d]", schedName, idx), sched)
		}

	}
}

// Configure reads the configuration and returns a new Checkable object
func (c *HeaterSchedules) Configure(data interface{}, i interface{}) error {
	l := logging.NewLogger("HeaterSchedules.Configure")

	err := config.NewMapstructureDecoder(c).Decode(data)
	if err != nil {
		return err
	}

	l.Trace().
		Msgf("%+v", c)

	return nil
}
