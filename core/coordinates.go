package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/kelvins/sunrisesunset"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
)

// Coordinates represents GPS coordinates using latitude and longitude
type Coordinates struct {
	Latitude  float64
	Longitude float64

	// Store previous sunrise/sunset values because it takes 10s on raspberry to compute...
	sunrise    time.Time
	sunset     time.Time
	lastUpdate time.Time
	// mutex ensures that only one thread at a time modifies private variables
	mutex *sync.Mutex
}

// NewLatitudeLongitude gets the latitude and longitude of a Home Assistant zone entity
func NewLatitudeLongitude(zoneName string) (Coordinates, error) {
	entity, err := httpclient.SimpleClientSingleton.GetEntity("zone", zoneName)
	if err != nil {
		return Coordinates{}, fmt.Errorf("Unable to get latitude and longitude, err=%v", err)
	}

	coords := Coordinates{
		Latitude:  entity.State.Attributes["latitude"].(float64),
		Longitude: entity.State.Attributes["longitude"].(float64),
		mutex:     &sync.Mutex{},
	}
	return coords, nil
}

// GetSunriseSunset gets sunrise and sunset times
func (c *Coordinates) GetSunriseSunset(noCache bool) (time.Time, time.Time, error) {
	now := time.Now().Local()

	if c.mutex == nil {
		return time.Time{}, time.Time{}, fmt.Errorf("Error, call NewLatitudeLongitude to create Coordinates")
	}

	l := logging.NewLogger("GetSunriseSunset")

	defer func() { // need anonymous func to have correct duration
		l.Trace().Str("duration", time.Now().Sub(now).String()).Msg("Time taken to get sunrise/sunset dates")
	}()

	// if set, returning saved values
	// update every 24 hours
	if !noCache && now.Sub(c.lastUpdate) < 24*time.Hour && !c.sunrise.IsZero() && !c.sunset.IsZero() {
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
	c.sunrise = sunrise
	c.sunset = sunset
	c.mutex.Unlock()

	c.lastUpdate = now
	return c.sunrise, c.sunset, nil
}

// IsDarkNow returns true if it's dark outside
func (c *Coordinates) IsDarkNow(offsetDawn, offsetDusk time.Duration) bool {
	now := time.Now()
	sunrise, sunset, _ := c.GetSunriseSunset(false)
	sunrise = sunrise.Add(offsetDawn)
	sunset = sunset.Add(offsetDusk)
	return now.Before(sunrise) || now.After(sunset)
}
