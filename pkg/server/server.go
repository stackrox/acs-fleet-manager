package server

import (
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

type Server interface {
	Run()
	Stop()
	Listen() (net.Listener, error)
	Serve(net.Listener)
}

func removeTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		next.ServeHTTP(w, r)
	})
}

// Exit on error
func check(err error, msg string, timeout time.Duration) {
	if err != nil && err != http.ErrServerClosed {
		glog.Errorf("%s: %s", msg, err)
		sentry.CaptureException(err)
		sentry.Flush(timeout)
		os.Exit(1)
	}
}
