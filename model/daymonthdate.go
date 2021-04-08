package model

import (
	"reflect"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

// DayMonthDate represents a date given the layout day/month or day-month
type DayMonthDate time.Time

// StringToHeaterDateDecodeHookFunc returns a func to decode a day/month or day-month formatted string into a HeaterDate
func StringToDayMonthDateDecodeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {

		//l := logging.NewLogger("StringToDayMonthDateDecodeHookFunc")

		if f.Kind() != reflect.String {
			//l.Debug().Msg("f data is not of type String")
			return data, nil
		}
		if t != reflect.TypeOf(DayMonthDate(time.Time{})) {
			//l.Debug().Msg("t data is not of type DayMonthDate")
			return data, nil
		}

		// Convert it by parsing
		layout := "02/01"
		if strings.Contains(data.(string), "-") { // Format is day/month or day-month
			ti, err := time.Parse(strings.Replace(layout, "/", "-", 1), data.(string))
			return DayMonthDate(ti), err
		}

		ti, err := time.Parse(layout, data.(string))
		//l.Debug().Msgf("t data is of type DayMonthDate, converting %s to %s", data.(string), ti.String())
		return DayMonthDate(ti), err
	}
}

// After returns true if d is After date (d year and hours are taken from given date)
func (d DayMonthDate) After(date time.Time) bool {
	dTime := time.Time(d)
	t := time.Date(date.Year(), dTime.Month(), dTime.Day(), date.Hour(), date.Minute(), date.Second(), date.Nanosecond(), date.Location())
	return t.After(date)
}

// Before returns true if d is Before date (d year and hours are taken from time.Now())
func (d DayMonthDate) Before(date time.Time) bool {
	dTime := time.Time(d)
	t := time.Date(date.Year(), dTime.Month(), dTime.Day(), date.Hour(), date.Minute(), date.Second(), date.Nanosecond(), date.Location())
	return t.Before(date)
}
