//go:build debug

package logger

const DEBUG = true

var (
	Debugln = func(args ...any) { logChan <- args }
)
