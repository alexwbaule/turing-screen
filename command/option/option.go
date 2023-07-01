package option

type StartMode byte
type FlipMode byte
type SleepInterval byte

type Option struct {
	Name    string
	Bytes   []byte
	Start   StartMode
	Padding byte
	Flip    FlipMode
	Sleep   SleepInterval
}

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

func SetOptions(s StartMode, f FlipMode, si SleepInterval) Option {
	return Option{
		Name: "OPTIONS",
		Bytes: []byte{
			0x7d, 0xef, 0x69, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x2d,
		},
		Start:   s,
		Padding: 0x00,
		Flip:    f,
		Sleep:   si,
	}
}

func (o Option) GetBytes() []byte {
	tmp := make([]byte, 250)
	cmd := append(o.Bytes, byte(o.Start), o.Padding, byte(o.Flip), byte(o.Sleep))
	copy(tmp, cmd)
	return tmp
}
