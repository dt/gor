package main

import (
	"time"
)

func DurationToMs(d time.Duration) int64 {
	sec := d / time.Second
	nsec := d % time.Second
	fl := float64(sec) + float64(nsec)*1e-6
	return int64(fl)
}
