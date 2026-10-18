// Package logger provides a minimal logging interface and zap implementation.
package logger

// Logger defines a minimal interface for structured logging.
type Logger interface {
	Info(msg string, args ...any)
	Fatal(msg string, args ...any)
}
