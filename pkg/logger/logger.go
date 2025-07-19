package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config represents logger configuration
type Config struct {
	Level  string `json:"level"`  // debug, info, warn, error
	Format string `json:"format"` // json, console
}

// NewLogger creates a new logger with the given configuration
func NewLogger(cfg *Config) (*zap.Logger, error) {
	level := zap.InfoLevel
	if cfg != nil {
		switch cfg.Level {
		case "debug":
			level = zap.DebugLevel
		case "info":
			level = zap.InfoLevel
		case "warn":
			level = zap.WarnLevel
		case "error":
			level = zap.ErrorLevel
		}
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg != nil && cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel)), nil
}

// DefaultLogger creates a new logger with default configuration
func DefaultLogger() *zap.Logger {
	logger, err := NewLogger(&Config{
		Level:  "info",
		Format: "json",
	})
	if err != nil {
		// If we can't create a logger, use a no-op logger
		return zap.NewNop()
	}
	return logger
}