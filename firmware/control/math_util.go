package control

func pow3(v int32) int32 {
	r := v * v / 256
	r = r * v / 256
	return r
}

func absInt32(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}
