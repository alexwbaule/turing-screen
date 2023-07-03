package utils

func IsInteger(val float64) bool {
	return val == float64(int(val))
}

func BZero(s int, b byte) []byte {
	tmp := make([]byte, s)
	for i := 0; i < s; i++ {
		tmp[i] = b
	}
	return tmp
}

func PadBegin(bb []byte, size int) []byte {
	l := len(bb)
	if l == size {
		return bb
	}
	if l > size {
		return bb
	}
	tmp := make([]byte, size)
	copy(tmp[size-l:], bb)
	return tmp
}
