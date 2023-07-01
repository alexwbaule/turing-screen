package option

type StartMode byte
type FlipMode byte
type SleepInterval byte

const (
	Default StartMode = 0x00
	Image   StartMode = 0x01
	Video   StartMode = 0x02
	NoFlip  FlipMode  = 0x00
	Flip180 FlipMode  = 0x01

	Disabled   SleepInterval = 0x00
	Interval01 SleepInterval = 0x01
	Interval02 SleepInterval = 0x02
	Interval03 SleepInterval = 0x03
	Interval04 SleepInterval = 0x04
	Interval05 SleepInterval = 0x05
	Interval06 SleepInterval = 0x06
	Interval07 SleepInterval = 0x07
	Interval08 SleepInterval = 0x08
	Interval09 SleepInterval = 0x09
	Interval10 SleepInterval = 0x0a
)

type Option struct {
	name    string
	bytes   []byte
	start   StartMode
	padding byte
	flip    FlipMode
	sleep   SleepInterval
}

func NewOption() *Option {
	return &Option{}
}

func (o *Option) SetOptions(s StartMode, f FlipMode, si SleepInterval) {
	o.name = "OPTIONS"
	o.bytes = []byte{
		0x7d, 0xef, 0x69, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x2d,
	}
	o.start = s
	o.padding = 0x00
	o.flip = f
	o.sleep = si
}

func (o *Option) GetBytes() []byte {
	tmp := make([]byte, 250)
	cmd := append(o.bytes, byte(o.start), o.padding, byte(o.flip), byte(o.sleep))
	copy(tmp, cmd)
	return tmp
}

func (o *Option) GetName() string {
	return o.name
}

func (o *Option) GetSize() int {
	return 0
}

func (o *Option) ValidateCommand([]byte, int) error {
	return nil
}
