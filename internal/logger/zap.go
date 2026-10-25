package logger

import "go.uber.org/zap"

type zapLogger struct {
	log *zap.SugaredLogger
}

// NewZap creates a production-configured zap logger with the specified log level.
func NewZap(level string) (Logger, error) {

	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}

	cfg := zap.NewProductionConfig()

	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return &zapLogger{log: zl.Sugar()}, nil
}

// Info logs an informational message with optional key-value pairs.
func (z *zapLogger) Info(msg string, args ...any) { z.log.Infow(msg, args...) }

// Error logs an error message.
func (z *zapLogger) Error(msg string) { z.log.Error(msg) }

// Debug logs a debug-level message.
func (z *zapLogger) Debug(msg string) { z.log.Debug(msg) }

// Fatal logs a fatal message with optional key-value pairs and exits the program.
func (z *zapLogger) Fatal(msg string, args ...any) { z.log.Fatalw(msg, args...) }

// With returns a new Logger with the specified key-value pairs attached
// to all future log entries.
func (z *zapLogger) With(args ...any) Logger {
	return &zapLogger{log: z.log.With(args...)}
}
