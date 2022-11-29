package log

import "context"

type ctxKey int

const (
	logKey ctxKey = iota
)

// ContextLogger extract logger facade from context.
func ContextLogger(ctx context.Context) Logger {
	if ctx.Value(logKey) == nil {
		return noopLogger
	}
	return ctx.Value(logKey).(Logger)
}

// ContextWithLogger puts logger facade into context.
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, logKey, logger)
}

// ContextWithStdoutLogger puts stdout logger facade into context.
func ContextWithStdoutLogger(ctx context.Context) context.Context {
	return context.WithValue(ctx, logKey, stdoutLogger)
}
