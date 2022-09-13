package integration

import (
	"flag"
	"os"
	"runtime"
	"testing"

	glog "github.com/stackrox/acs-fleet-manager/pkg/logging"
)

func TestMain(m *testing.M) {
	flag.Parse()
	glog.V(10).Infof("Starting integration test using go version %s", runtime.Version())
	os.Exit(m.Run())
}

// skip integration tests whose implementation is not fully implemented
// https://bf2.zulipchat.com/#narrow/stream/315461-factorized-fleet-manager/topic/Integration.20tests
func skipNotFullyImplementedYet(t *testing.T) {
	t.Skip("Not fully implemented yet")
}
