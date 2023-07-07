package entity

import "time"

type Network struct {
	Interval time.Duration
	Wired
	Wifi
}

type Wired struct {
	Interval time.Duration
	Upload
	Download
}
type Wifi struct {
	Interval time.Duration
	Upload
	Download
}

type Upload struct {
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}

type Download struct {
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}
