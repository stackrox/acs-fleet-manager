// Package profiler provides profiling tools for debugging.
package profiler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"sync"

	"github.com/golang/glog"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
)

var _ server.Server = &PprofServer{}
var _ environments.BootService = &PprofServer{}

// PprofServer ...
type PprofServer struct {
	httpServer *http.Server
}

// Start ...
func (p *PprofServer) Start() {
	go p.Run()
}

var (
	oncePprofServer     sync.Once
	pprofServerInstance *PprofServer
)

// SingletonPprofServer returns the PprofServer
func SingletonPprofServer() *PprofServer {
	oncePprofServer.Do(func() {
		router := mux.NewRouter()
		router.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		router.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		router.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		router.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		router.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		router.Handle("/debug/pprof/{cmd}", http.HandlerFunc(pprof.Index)) // special handling for Gorilla mux
		httpServer := &http.Server{
			Addr:    "localhost:6060",
			Handler: router,
		}

		pprofServerInstance = &PprofServer{
			httpServer: httpServer,
		}
	})
	return pprofServerInstance
}

// Stop ...
func (p *PprofServer) Stop() {
	err := p.httpServer.Shutdown(context.Background())
	if err != nil {
		glog.Warningf("Unable to stop profiling server: %s", err)
	}
}

// Listen ...
func (p *PprofServer) Listen() (net.Listener, error) {
	return nil, nil
}

// Serve ...
func (p *PprofServer) Serve(listener net.Listener) {
}

// Run ...
func (p *PprofServer) Run() {
	err := p.httpServer.ListenAndServe()
	if err != nil {
		panic(fmt.Sprintf("starting pprof server failed %s", err))
	}
}
