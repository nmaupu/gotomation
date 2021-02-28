package app

import "time"

// DateIsToday returns true if the given date happens today, false otherwise
func DateIsToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
}
