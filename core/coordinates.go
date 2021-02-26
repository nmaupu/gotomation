package core

import (
	"fmt"
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
	coords coordinates
	once   sync.Once
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
}

// InitCoordinates gets the latitude and longitude of a Home Assistant zone entity
func InitCoordinates(zoneName string) error {
	var err error
	once.Do(
		func() {
			entity, err := httpclient.GetSimpleClient().GetEntity("zone", zoneName)
			if err != nil {
				err = errors.Wrapf(err, "Unable to get latitude and longitude, err=%v", err)
			} else {
				coords = coordinates{
					Latitude:          entity.State.Attributes["latitude"].(float64),
					Longitude:         entity.State.Attributes["longitude"].(float64),
					mutex:             &sync.Mutex{},
					sunriseSunsetDone: make(chan bool, 1),
				}
			}
		})

	return err
}

// Coords returns the Coordinates singleton
func Coords() Coordinates {
	return &coords
}

// StopSunriseSunset stops the sunrise/sunset refresh goroutine
func (c *coordinates) Stop() {
	c.sunriseSunsetDone <- true
}

func (c *coordinates) Start() error {
	l := logging.NewLogger("Coordinates.Start")

	// first init before ticker ticks
	app.RoutinesWG.Add(1)
	go func() {
		defer app.RoutinesWG.Done()
		c.getSunriseSunset(true)
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
				c.getSunriseSunset(true)
			}
		}
	}()

	return nil
}

func (c *coordinates) GetLatitude() float64 {
	return c.Latitude
}

func (c *coordinates) GetLongitude() float64 {
	return c.Longitude
}

func (c *coordinates) GetSunriseSunset() (time.Time, time.Time, error) {
	return c.getSunriseSunset(false)
}

// GetSunriseSunset gets sunrise and sunset times
func (c *coordinates) getSunriseSunset(noCache bool) (time.Time, time.Time, error) {
	now := time.Now().Local()

	if c.mutex == nil {
		return time.Time{}, time.Time{}, fmt.Errorf("Error, call NewLatitudeLongitude to create Coordinates")
	}

	l := logging.NewLogger("GetSunriseSunset")

	defer func() { // need anonymous func to have correct duration
		l.Trace().Str("duration", time.Now().Sub(now).String()).Msg("Time taken to get sunrise/sunset dates")
	}()

	// if set, returning saved values
	if !noCache && now.Sub(c.lastUpdate) < 12*time.Hour && !c.sunrise.IsZero() && !c.sunset.IsZero() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		return c.sunrise, c.sunset, nil
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

	c.mutex.Lock()
	c.sunrise = time.Date(now.Year(), now.Month(), now.Day(), sunrise.Hour(), sunrise.Minute(), sunrise.Second(), sunrise.Nanosecond(), sunrise.Location())
	c.sunset = time.Date(now.Year(), now.Month(), now.Day(), sunset.Hour(), sunset.Minute(), sunset.Second(), sunset.Nanosecond(), sunset.Location())
	c.mutex.Unlock()

	c.lastUpdate = now
	return c.sunrise, c.sunset, nil
}

// IsDarkNow returns true if it's dark outside
func (c *coordinates) IsDarkNow(offsetDawn, offsetDusk time.Duration) bool {
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
