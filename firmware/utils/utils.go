package utils

func Map(in_min, in_max, out_min, out_max int32) func(x int32) int32 {
	return func(x int32) int32 {
		return int32(int64(x-in_min)*int64(out_max-out_min)/int64(in_max-in_min)) + out_min
	}
}

func Limit(min, max int32) func(x int32) int32 {
	return func(x int32) int32 {
		switch {
		case x > max:
			return max
		case x < min:
			return min
		}
		return x
	}
}
