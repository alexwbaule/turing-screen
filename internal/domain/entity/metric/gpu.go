package metric

type GPU struct {
	Temperature uint64 // °C
	Load        uint64 // percentage
	Power       uint64 // watts
	Frequency   uint64 // MHz (core clock)
	MemClock    uint64 // MHz (memory clock)
	Voltage     uint64 // mV
	VRAMUsage   uint64 // bytes
	VRAMSize    uint64 // bytes
	Fan         uint64 // RPM
}
