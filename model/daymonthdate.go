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

		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(DayMonthDate(time.Time{})) {
			return data, nil
		}

		// Convert it by parsing
		layout := "02/01"
		if strings.Contains(data.(string), "-") { // Format is day/month or day-month
			return time.Parse(strings.Replace(layout, "/", "-", 1), data.(string))
		}

		return time.Parse(layout, data.(string))
	}
}

// After returns true if d is After date (d year and hour are taken from time.Now())
func (d DayMonthDate) After(date time.Time) bool {
	now := time.Now()
	dTime := time.Time(d)
	t := time.Date(now.Year(), dTime.Month(), dTime.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())
	return t.After(date)
}

// Before returns true if d is Before date (d year and hours are taken from time.Now())
func (d DayMonthDate) Before(date time.Time) bool {
	now := time.Now()
	dTime := time.Time(d)
	t := time.Date(now.Year(), dTime.Month(), dTime.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())
	return t.Before(date)
}
