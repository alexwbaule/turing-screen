package entity

import "time"

type Memory struct {
	Interval         time.Duration
	StatTexts        map[string]StatText
	StatProgressBars map[string]StatProgressBar
	StatRadialBars   map[string]StatRadialBar
}
