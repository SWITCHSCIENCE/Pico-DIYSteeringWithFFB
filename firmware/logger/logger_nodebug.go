//go:build !debug

package logger

const DEBUG = false

var (
	Debugln = func(args ...any) {}
)
