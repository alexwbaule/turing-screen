package entity

import "time"

type GPU struct {
	Interval         time.Duration
	StatTexts        map[string]StatText
	StatProgressBars map[string]StatProgressBar
	StatRadialBars   map[string]StatRadialBar
}
