package config

import (
	"strconv"
	"strings"
	"time"
)

const hoursPerDay = 24

// ParseElapsedDuration parses Go duration strings and elapsed day values.
func ParseElapsedDuration(value string) (time.Duration, error) {
	// time.ParseDuration supports elapsed hours but has no day unit.
	if daysText, ok := strings.CutSuffix(value, "d"); ok {
		days, err := strconv.ParseFloat(daysText, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(days * float64(hoursPerDay*time.Hour)), nil
	}
	return time.ParseDuration(value)
}
