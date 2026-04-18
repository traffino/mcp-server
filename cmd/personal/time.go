package main

import (
	"fmt"
	"time"
)

// appLoc holds the application timezone, set at bootstrap via PERSONAL_TZ.
var appLoc = time.UTC

func initTimezone(tzName string) error {
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", tzName, err)
	}
	appLoc = loc
	return nil
}

func today() string {
	return time.Now().In(appLoc).Format("2006-01-02")
}

func nowString() string {
	return time.Now().In(appLoc).Format("2006-01-02 15:04:05")
}

func currentYear() int {
	return time.Now().In(appLoc).Year()
}
