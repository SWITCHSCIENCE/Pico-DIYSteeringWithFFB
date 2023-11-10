package logger

import "encoding/hex"

var logChan = make(chan []any, 16)

func log(args ...any) {
	for i, v := range args {
		switch vv := v.(type) {
		case []byte:
			print(hex.EncodeToString(vv))
		default:
			print(v)
		}
		if i == len(args)-1 {
			println()
		} else {
			print(" ")
		}
	}
}

func init() {
	go func() {
		for v := range logChan {
			log(v...)
		}
	}()
}
