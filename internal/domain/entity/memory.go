package entity

import "time"

type Memory struct {
	Interval time.Duration
	MemorySwap
	MemoryPercent
	MemoryUsed
	MemoryFree
}

type MemorySwap struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}

type MemoryPercent struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}

type MemoryUsed struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}

type MemoryFree struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}
