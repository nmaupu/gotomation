package core

import (
	"fmt"
	"time"

	"github.com/kelvins/sunrisesunset"
	"github.com/nmaupu/gotomation/httpclient"
	"github.com/nmaupu/gotomation/logging"
)

// Coordinates represents GPS coordinates using latitude and longitude
type Coordinates struct {
	Latitude  float64
	Longitude float64
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
	}
	return coords, nil
}

// GetSunriseSunset gets sunrise and sunset times
func (c Coordinates) GetSunriseSunset() (time.Time, time.Time, error) {
	l := logging.NewLogger("GetSunriseSunset")

	// Testing sunrise/sunset
	now := time.Now()
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

	return sunrise, sunset, nil
}

// IsDarkNow returns true if it's dark outside
func (c Coordinates) IsDarkNow(offsetDawn, offsetDusk time.Duration) bool {
	now := time.Now()
	sunrise, sunset, _ := c.GetSunriseSunset()
	sunrise = sunrise.Add(offsetDawn)
	sunset = sunset.Add(offsetDusk)
	return now.Before(sunrise) || now.After(sunset)
}
