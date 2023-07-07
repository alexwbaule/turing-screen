package entity

import "time"

type GPU struct {
	Interval time.Duration
	GPUPercentage
	GPUMemory
	GPUTemperature
}

type GPUPercentage struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}

type GPUMemory struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}

type GPUTemperature struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}
