package entity

import "time"

type DateTime struct {
	Interval time.Duration
	Date     StatText
	Time     StatText
}
