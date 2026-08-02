package logger

import "go.uber.org/zap"

type zapLogger struct {
	log *zap.SugaredLogger
}

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

func (z *zapLogger) Info(msg string, args ...any)  { z.log.Infow(msg, args...) }
func (z *zapLogger) Error(msg string)              { z.log.Error(msg) }
func (z *zapLogger) Debug(msg string)              { z.log.Debug(msg) }
func (z *zapLogger) Fatal(msg string, args ...any) { z.log.Fatalw(msg, args...) }

func (z *zapLogger) With(args ...any) Logger {
	return &zapLogger{log: z.log.With(args...)}
}
