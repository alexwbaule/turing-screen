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
