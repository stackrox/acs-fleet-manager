// Kube controller-runtime client uses structured logr.Logger under the hood.
// ACSCS services use glog unstructured logging.
// This file contains a wrapper which helps to send structured logr.Logger logs to glog logSink.
// This wrapper can be removed once ACSCS service move to structured logging instead of glog

package logger

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/golang/glog"
	"strings"
)

const msgKeyPattern = "msg\"="

// NewKubeAPILogger creates a new logr.Logger instance which uses a glog.Warning as log message sink.
// This logger should be passed to controller-runtime client. The client will use it to print log messages.
func NewKubeAPILogger() logr.Logger {
	return funcr.New(func(prefix, args string) {
		logMsg := sanitizeLog(args)
		if prefix != "" {
			glog.Warningf("%s: %s\n", prefix, logMsg)
		} else {
			glog.Warningln(logMsg)
		}
	}, funcr.Options{})
}

// sanitizeLog removes redundant builtin logr.Logger log keys from the log message.
// Only `msg` value is worth to log eventually
func sanitizeLog(log string) string {
	at := strings.Index(log, msgKeyPattern)
	if at > 0 {
		return log[at+len(msgKeyPattern):]
	}
	return log
}
