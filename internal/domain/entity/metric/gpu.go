package metric

type GPU struct {
	Temperature uint64 // °C
	Load        uint64 // percentage
	Power       uint64 // watts
	VRAMUsage   uint64 // bytes
	VRAMSize    uint64 // bytes
}
