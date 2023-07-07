package entity

import "time"

type Disk struct {
	Interval time.Duration
	DiskUsed
	DiskTotal
	DiskFree
}

type DiskUsed struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}

type DiskTotal struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}

type DiskFree struct {
	Interval        time.Duration
	StatText        StatText
	StatProgressBar StatProgressBar
	StatRadialBar   StatRadialBar
}
