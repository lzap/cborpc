package log

import (
	"fmt"
)

type Level int

const (
	TRC  = Level(256)
	DBG  = Level(128)
	INF  = Level(64)
	WRN  = Level(32)
	ERR  = Level(16)
	NONE = Level(8)
)

// Logger facade for integration with other logging solutions.
type Logger interface {
	Msgf(level Level, format string, values ...any)
}

type noop struct{}

func (_ noop) Msgf(_ Level, _ string, _ ...any) {
}

var noopLogger = &noop{}

type stdout struct{}

func (_ stdout) Msgf(_ Level, format string, values ...any) {
	fmt.Printf(format, values...)
	fmt.Println("")
}

var stdoutLogger = &stdout{}
