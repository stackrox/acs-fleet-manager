package db

import (
	"context"
	"strings"
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"gorm.io/gorm/logger"
)

type customLoggerWithMetricsCollector struct {
}

// LogMode sets the log level
func (l customLoggerWithMetricsCollector) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

// Info print information
func (l customLoggerWithMetricsCollector) Info(ctx context.Context, msg string, data ...interface{}) {
}

// Warn print warning message
func (l customLoggerWithMetricsCollector) Warn(ctx context.Context, msg string, data ...interface{}) {
}

// Error print error message
func (l customLoggerWithMetricsCollector) Error(ctx context.Context, msg string, data ...interface{}) {
}

// Trace trace the sql query and its execution time.
func (l customLoggerWithMetricsCollector) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	status := "success"

	if err != nil {
		status = "failure"
	}

	sql, _ := fc()
	sql = strings.TrimLeft(sql, " ")
	tokens := strings.Split(sql, " ")
	metrics.IncreaseDatabaseQueryCount(status, tokens[0])
	metrics.UpdateDatabaseQueryDurationMetric(status, tokens[0], elapsed)
}
