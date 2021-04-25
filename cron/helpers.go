package cron

func createFnWithTimezone(timezone string, fn func(timezone string)) func() {
	return func() {
		fn(timezone)
	}
}
