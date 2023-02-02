package logger

import (
	"context"
	"testing"
)

func TestIt(t *testing.T) {
	log := NewUHCLogger(context.TODO())
	var ct int32
	ct = 25
	log.InfoChangedInt32(&ct, "log message")
}
