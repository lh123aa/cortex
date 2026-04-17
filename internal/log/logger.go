package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Logger is the global logger instance
	Logger *zap.Logger
)

func init() {
	// Initialize with production config (JSON output to stderr)
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"stderr"}
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, _ := cfg.Build()
	Logger = logger
}

// NewLogger creates a new logger with custom configuration
func NewLogger(level zapcore.Level) *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.OutputPaths = []string{"stderr"}
	logger, _ := cfg.Build()
	return logger
}

// Sync flushes any buffered log entries
func Sync() {
	if Logger != nil {
		Logger.Sync()
	}
}
