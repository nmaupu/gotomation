package core

import (
	"sort"
	"strings"
	"time"

	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/model/config"
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
	Scheds     map[SchedulesDays][]HeaterSchedule `mapstructure:"schedules"`
	DefaultEco float64                            `mapstructure:"default_eco"`
}

// HeaterSchedule represents a heater's schedule
type HeaterSchedule struct {
	beg     time.Time `mapstructure:"beg"`
	end     time.Time `mapstructure:"end"`
	Confort float64   `mapstructure:"confort"`
	Eco     float64   `mapstructure:"eco"`
}

func getTodayTime(t time.Time, loc *time.Location) time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

// Beg returns c.Beg time with today's date
func (c HeaterSchedule) Beg(loc *time.Location) time.Time {
	return getTodayTime(c.beg, loc)
}

// End returns c.End time with today's date
func (c HeaterSchedule) End(loc *time.Location) time.Time {
	return getTodayTime(c.end, loc)
}

// IsActive returns true if given 't' is between c.Beg and c.End
func (c HeaterSchedule) IsActive(t time.Time) bool {
	return t.After(c.Beg(t.Location())) && t.Before(c.End(t.Location()))
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
		if strings.ToLower(strings.Trim(s, " ")) == strings.ToLower(v) {
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
			return scheds[i].end.Before(scheds[j].beg)
		})

		c.Scheds[k] = scheds
	}
}

// GetTemperatureToSet returns the temperature to set corresponding to the time given in parameter
func (c *HeaterSchedules) GetTemperatureToSet(t time.Time) float64 {
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
				return sched.Confort
			}

			if t.After(sched.End(t.Location())) {
				finalTemp = sched.Eco
			}
		}
	}

	return finalTemp
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
