package handlers

import (
	"context"
	"fmt"
	"os"

	log "github.com/stackrox/acs-fleet-manager/pkg/logging"
)

// LoggerBuilder contains the configuration and logic needed to build a logger that uses the internal
// `logging` package. Don't create instances of this type directly, use the NewLoggerBuilder function
// instead.
type LoggerBuilder struct {
	debugEnabled bool
	infoEnabled  bool
	warnEnabled  bool
	errorEnabled bool
	logger       *log.Logger
}

// Logger is a logger that uses the Go `log` package.
type Logger struct {
	debugEnabled bool
	infoEnabled  bool
	warnEnabled  bool
	errorEnabled bool
	logger       *log.Logger
}

// NewLoggerBuilder creates a builder that knows how to build a logger that uses the internal logging
// package. By default these loggers will have enabled the information, warning and error levels
func NewLoggerBuilder(logger *log.Logger) *LoggerBuilder {
	// Allocate the object:
	builder := new(LoggerBuilder)

	// Set default values:
	builder.debugEnabled = false
	builder.infoEnabled = true
	builder.warnEnabled = true
	builder.errorEnabled = true
	builder.logger = logger

	return builder
}

// Debug enables or disables the debug level.
func (b *LoggerBuilder) Debug(flag bool) *LoggerBuilder {
	b.debugEnabled = flag
	return b
}

// Info enables or disables the information level.
func (b *LoggerBuilder) Info(flag bool) *LoggerBuilder {
	b.infoEnabled = flag
	return b
}

// Warn enables or disables the warning level.
func (b *LoggerBuilder) Warn(flag bool) *LoggerBuilder {
	b.warnEnabled = flag
	return b
}

// Error enables or disables the error level.
func (b *LoggerBuilder) Error(flag bool) *LoggerBuilder {
	b.errorEnabled = flag
	return b
}

// Build creates a new logger using the configuration stored in the builder.
func (b *LoggerBuilder) Build() (logger *Logger, err error) {
	// Allocate and populate the object:
	logger = new(Logger)
	logger.debugEnabled = b.debugEnabled
	logger.infoEnabled = b.infoEnabled
	logger.warnEnabled = b.warnEnabled
	logger.errorEnabled = b.errorEnabled
	logger.logger = b.logger

	return
}

// DebugEnabled returns true iff the debug level is enabled.
func (l *Logger) DebugEnabled() bool {
	return l.debugEnabled
}

// InfoEnabled returns true iff the information level is enabled.
func (l *Logger) InfoEnabled() bool {
	return l.infoEnabled
}

// WarnEnabled returns true iff the warning level is enabled.
func (l *Logger) WarnEnabled() bool {
	return l.warnEnabled
}

// ErrorEnabled returns true iff the error level is enabled.
func (l *Logger) ErrorEnabled() bool {
	return l.errorEnabled
}

// Debug sends to the log a debug message formatted using the fmt.Sprintf function and the given
// format and arguments.
func (l *Logger) Debug(ctx context.Context, format string, args ...interface{}) {
	if l.debugEnabled {
		msg := fmt.Sprintf(format, args...)
		// #nosec G104
		l.logger.Debug(msg)
	}
}

// Info sends to the log an information message formatted using the fmt.Sprintf function and the
// given format and arguments.
func (l *Logger) Info(ctx context.Context, format string, args ...interface{}) {
	if l.infoEnabled {
		msg := fmt.Sprintf(format, args...)
		// #nosec G104
		l.logger.Info(msg)
	}
}

// Warn sends to the log a warning message formatted using the fmt.Sprintf function and the given
// format and arguments.
func (l *Logger) Warn(ctx context.Context, format string, args ...interface{}) {
	if l.warnEnabled {
		msg := fmt.Sprintf(format, args...)
		// #nosec G104
		l.logger.Warn(msg)
	}
}

// Error sends to the log an error message formatted using the fmt.Sprintf function and the given
// format and arguments.
func (l *Logger) Error(ctx context.Context, format string, args ...interface{}) {
	if l.errorEnabled {
		msg := fmt.Sprintf(format, args...)
		// #nosec G104
		l.logger.Error(msg)
	}
}

// Fatal sends to the log an error message formatted using the fmt.Sprintf function and the given
// format and arguments. After that it will os.Exit(1)
// This level is always enabled
func (l *Logger) Fatal(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// #nosec G104
	l.logger.Fatal(msg)
	os.Exit(1)
}
