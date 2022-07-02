package core

import (
	"fmt"
	"github.com/nmaupu/gotomation/model"
	"sync"
	"time"

	"github.com/kelvins/sunrisesunset"
	"github.com/nmaupu/gotomation/app"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
	"github.com/nmaupu/gotomation/routines"
	"github.com/pkg/errors"
)

var (
	coords  coordinates
	once    sync.Once
	onceErr error
)

// Coordinates represents GPS coordinates using latitude and longitude
type Coordinates interface {
	routines.Runnable
	GetSunriseSunset() (time.Time, time.Time, error)
	IsDarkNow(offsetDawn, offsetDusk time.Duration) bool
	GetLatitude() float64
	GetLongitude() float64
}

// Coordinates represents GPS coordinates using latitude and longitude
type coordinates struct {
	Latitude  float64
	Longitude float64

	// Store previous sunrise/sunset values because it takes 10s on raspberry to compute...
	sunrise    time.Time
	sunset     time.Time
	lastUpdate time.Time
	// mutex ensures that only one thread at a time modifies private variables
	mutex             *sync.Mutex
	sunriseSunsetDone chan bool

	started        bool
	mutexStopStart sync.Mutex
}

// InitCoordinates gets the latitude and longitude of a Home Assistant zone entity
func InitCoordinates(zoneName string) error {
	once.Do(
		func() {
			var entity model.HassEntity
			entity, onceErr = httpclient.GetSimpleClient().GetEntity("zone", zoneName)
			if onceErr != nil {
				onceErr = errors.Wrapf(onceErr, "Unable to get latitude and longitude")
			} else {
				coords = coordinates{
					Latitude:          entity.State.Attributes["latitude"].(float64),
					Longitude:         entity.State.Attributes["longitude"].(float64),
					mutex:             &sync.Mutex{},
					sunriseSunsetDone: make(chan bool, 1),
				}
			}
		})

	return onceErr
}

// Coords returns the Coordinates singleton
func Coords() Coordinates {
	return &coords
}

// Stop stops the sunrise/sunset refresh goroutine
func (c *coordinates) Stop() {
	c.mutexStopStart.Lock()
	defer c.mutexStopStart.Unlock()
	if !c.started {
		return
	}
	c.sunriseSunsetDone <- true
	c.started = false
}

func (c *coordinates) Start() error {
	c.mutexStopStart.Lock()
	defer c.mutexStopStart.Unlock()
	if c.started {
		return nil
	}

	// init stop channel
	c.sunriseSunsetDone = make(chan bool, 1)

	l := logging.NewLogger("Coordinates.Start")

	// first init before ticker ticks
	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		c.getSunriseSunset(false)
	}()

	app.RoutinesWG.Add(1)
	go func() { // updating sunrise / sunset dates once in a while
		defer app.RoutinesWG.Done()
		l.Debug().Msg("Starting sunrise/sunset refresh go routine")
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-c.sunriseSunsetDone:
				l.Trace().Msg("Exiting sunrise/sunset refresh go routine")
				return
			case <-ticker.C:
				c.getSunriseSunset(false)
			}
		}
	}()

	c.started = true
	return nil
}

func (c *coordinates) IsStarted() bool {
	c.mutexStopStart.Lock()
	defer c.mutexStopStart.Unlock()
	return c.started && c.sunriseSunsetDone != nil
}

func (c *coordinates) GetLatitude() float64 {
	return c.Latitude
}

func (c *coordinates) GetLongitude() float64 {
	return c.Longitude
}

func (c *coordinates) GetSunriseSunset() (time.Time, time.Time, error) {
	return c.getSunriseSunset(true)
}

// GetSunriseSunset gets sunrise and sunset times
func (c *coordinates) getSunriseSunset(cache bool) (time.Time, time.Time, error) {
	if c.mutex == nil {
		return time.Time{}, time.Time{}, fmt.Errorf("call NewLatitudeLongitude to create Coordinates")
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now().Local()

	l := logging.NewLogger("GetSunriseSunset")

	defer func() { // need anonymous func to have correct duration
		l.Trace().Str("duration", time.Now().Sub(now).String()).Msg("Time taken to get sunrise/sunset dates")
	}()

	// If cache is true, return values if freshness is < 12 hours
	// If func has been called too soon (< 5 secs), returning cached values if they exist even if cache is false
	if !c.sunrise.IsZero() && !c.sunset.IsZero() {
		durationSinceLastUpdate := now.Sub(c.lastUpdate)
		if (cache && durationSinceLastUpdate < 12*time.Hour) ||
			durationSinceLastUpdate < 30*time.Second {
			return c.sunrise, c.sunset, nil
		}
	}

	name, offset := now.Zone()
	l.Trace().Str("zone_name", name).Int("offset", offset).Msg("Timezone information")

	p := sunrisesunset.Parameters{
		Latitude:  c.Latitude,
		Longitude: c.Longitude,
		UtcOffset: float64(offset) / 3600,
		Date:      now,
	}

	sunrise, sunset, err := p.GetSunriseSunset()
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	// Using now.Location() to get correct timezone (sunrise and sunset doesn't have a Location set to return UTC time.UTC)
	c.sunrise = time.Date(now.Year(), now.Month(), now.Day(), sunrise.Hour(), sunrise.Minute(), sunrise.Second(), sunrise.Nanosecond(), now.Location())
	c.sunset = time.Date(now.Year(), now.Month(), now.Day(), sunset.Hour(), sunset.Minute(), sunset.Second(), sunset.Nanosecond(), now.Location())
	c.lastUpdate = now

	l.Info().
		Time("sunrise", c.sunrise).
		Time("sunset", c.sunset).
		Msg("Sunrise, sunset dates has been initialized successfully")

	return c.sunrise, c.sunset, nil
}

// IsDarkNow returns true if it's dark outside
func (c *coordinates) IsDarkNow(offsetDawn, offsetDusk time.Duration) bool {
	l := logging.NewLogger("IsDarkNow")
	if c.GetLatitude() == 0 || c.GetLongitude() == 0 {
		l.Warn().Msg("Latitude or longitude is not set, cannot determine if it's dark")
		return false
	}
	now := time.Now().Local()
	sunrise, sunset, _ := c.GetSunriseSunset()
	sunrise = sunrise.Add(offsetDawn)
	sunset = sunset.Add(offsetDusk)
	return now.Before(sunrise) || now.After(sunset)
}

// GetName returns the name of this runnable object
func (c *coordinates) GetName() string {
	return "Coordinates"
}

func (c *coordinates) IsAutoStart() bool {
	return true
}
