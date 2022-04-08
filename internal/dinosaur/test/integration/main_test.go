package integration

import (
	"flag"
	"os"
	"runtime"
	"testing"

	"github.com/golang/glog"
)

func TestMain(m *testing.M) {
	flag.Parse()
	glog.V(10).Infof("Starting integration test using go version %s", runtime.Version())
	os.Exit(m.Run())
}
