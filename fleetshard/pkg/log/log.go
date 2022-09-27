package log

import (
	"fmt"

	"github.com/golang/glog"
)

const (
	debugLogLevel = 10
)

// Debugf writes debug log messages.
func Debugf(format string, args ...interface{}) {
	// To avoid log entry point to this function, instruct glog to skip one
	// extra stack frame.
	//
	// Note that `Verbose.InfoDepth()` is unavailable in "golang/glog" at the
	// moment of writing. It is added to our private "glog" fork.

	msg := fmt.Sprintf(format, args)
	glog.V(debugLogLevel).InfoDepth(1, msg)
}
