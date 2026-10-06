package logger

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	global   *zap.Logger
	initOnce sync.Once
	level    zap.AtomicLevel
)

func Init(_ context.Context, logLevel string, asJSON, _ bool) error {
	initOnce.Do(func() {
		level = zap.NewAtomicLevelAt(parseLevel(logLevel))

		encoderCfg := zapcore.EncoderConfig{
			TimeKey:      "timestamp",
			LevelKey:     "level",
			MessageKey:   "message",
			CallerKey:    "caller",
			LineEnding:   zapcore.DefaultLineEnding,
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		}

		var encoder zapcore.Encoder
		if asJSON {
			encoder = zapcore.NewJSONEncoder(encoderCfg)
		} else {
			encoder = zapcore.NewConsoleEncoder(encoderCfg)
		}

		core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
		global = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	})

	if global == nil {
		return fmt.Errorf("logger init failed")
	}

	return nil
}

type Key string

const (
	traceIDKey Key = "trace_id"
	userIDKey  Key = "user_id"
)

type ctxLogger struct {
	base *zap.Logger
}

func Logger() *ctxLogger {
	if global == nil {
		return &ctxLogger{base: zap.NewNop()}
	}

	return &ctxLogger{base: global}
}

func (l *ctxLogger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	if l == nil || l.base == nil {
		return
	}
	allFields := append(fieldsFromContext(ctx), fields...)
	l.base.Debug(msg, allFields...)
}

func (l *ctxLogger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	if l == nil || l.base == nil {
		return
	}
	allFields := append(fieldsFromContext(ctx), fields...)
	l.base.Info(msg, allFields...)
}

func (l *ctxLogger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	if l == nil || l.base == nil {
		return
	}
	allFields := append(fieldsFromContext(ctx), fields...)
	l.base.Warn(msg, allFields...)
}

func (l *ctxLogger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	if l == nil || l.base == nil {
		return
	}
	allFields := append(fieldsFromContext(ctx), fields...)
	l.base.Error(msg, allFields...)
}

func Info(_ context.Context, msg string, fields ...zap.Field) {
	if global != nil {
		global.Info(msg, fields...)
	}
}

func Debug(_ context.Context, msg string, fields ...zap.Field) {
	if global != nil {
		global.Debug(msg, fields...)
	}
}

func Warn(_ context.Context, msg string, fields ...zap.Field) {
	if global != nil {
		global.Warn(msg, fields...)
	}
}

func Error(_ context.Context, msg string, fields ...zap.Field) {
	if global != nil {
		global.Error(msg, fields...)
	}
}

func parseLevel(levelStr string) zapcore.Level {
	switch levelStr {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func fieldsFromContext(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}

	fields := make([]zap.Field, 0, 2)
	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		fields = append(fields, zap.String(string(traceIDKey), traceID))
	}
	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		fields = append(fields, zap.String(string(userIDKey), userID))
	}

	return fields
}
