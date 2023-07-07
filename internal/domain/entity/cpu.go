package entity

import (
	"time"
)

type CPU struct {
	Interval *time.Duration
	*CPUPercentage
	*CPUFrequency
	*CPUTemperature
	*LoadAvg
}

type CPUPercentage struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}

type CPUFrequency struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}

type LoadAvg struct {
	One     StatText
	Five    StatText
	Fifteen StatText
}

type CPUTemperature struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}
