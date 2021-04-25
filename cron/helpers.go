package cron

import (
	"time"
)

func createFnWithTimezone(timezone string, fn func(location *time.Location)) func() {
	tz, err := time.LoadLocation(timezone)
	if err != nil {
		tz = time.UTC
	}
	return func() {
		fn(tz)
	}
}
