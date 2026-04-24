// Package logger provides a structured, leveled logger built on top of go.uber.org/zap.
// It is the single logging entrypoint for the entire application.
package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a production-ready zap.Logger.
// In development mode (env != "production"), it uses a human-readable console encoder.
func New(level, env string) (*zap.Logger, error) {
	logLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	var cfg zap.Config

	if env == "production" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.EncoderConfig.MessageKey = "message"
		cfg.EncoderConfig.LevelKey = "severity"
		cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		cfg.OutputPaths = []string{"stdout"}
		cfg.ErrorOutputPaths = []string{"stderr"}
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
	}

	cfg.Level = zap.NewAtomicLevelAt(logLevel)

	logger, err := cfg.Build(
		zap.AddCallerSkip(0),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(
			zap.String("service", os.Getenv("APP_NAME")),
			zap.String("version", os.Getenv("APP_VERSION")),
			zap.String("env", env),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("logger: build failed: %w", err)
	}

	return logger, nil
}

// MustNew is like New but panics on error. Suitable for main().
func MustNew(level, env string) *zap.Logger {
	l, err := New(level, env)
	if err != nil {
		panic(fmt.Sprintf("logger: %v", err))
	}
	return l
}

func parseLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info", "":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "dpanic":
		return zapcore.DPanicLevel, nil
	case "panic":
		return zapcore.PanicLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown log level: %q", level)
	}
}
