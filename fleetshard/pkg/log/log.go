package log

import "github.com/golang/glog"

const (
	debugLogLevel = 10
)

// Info write info messages.
func Info(args ...interface{}) {
	glog.Info(args)
}

// Infof write info messages.
func Infof(format string, args ...interface{}) {
	glog.Infof(format, args)
}

// Debug writes debug log messages.
func Debug(args ...interface{}) {
	glog.V(debugLogLevel).Info(args)
}

// Debugf writes debug log messages.
func Debugf(format string, args ...interface{}) {
	glog.V(debugLogLevel).Infof(format, args)
}

// Error logs to the ERROR, WARNING, and INFO logs.
func Error(args ...interface{}) {
	glog.Error(args)
}

// Errorf logs to the ERROR, WARNING, and INFO logs.
func Errorf(format string, args ...interface{}) {
	glog.Error(format, args)
}

// Warning logs to the WARNING and INFO logs.
func Warning(args interface{}) {
	glog.Warning(args)
}
