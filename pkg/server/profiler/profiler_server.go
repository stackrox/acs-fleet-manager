package profiler

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"net"
	"net/http"
	"net/http/pprof"
	"sync"
)

var _ server.Server = &PprofServer{}
var _ environments.BootService = &PprofServer{}

type PprofServer struct{}

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
		pprofServerInstance = &PprofServer{}
	})
	return pprofServerInstance
}

func (p *PprofServer) Stop() {
}

func (p *PprofServer) Listen() (net.Listener, error) {
	return nil, nil
}

func (p *PprofServer) Serve(listener net.Listener) {
}

func (p *PprofServer) Run() {
	router := mux.NewRouter()
	router.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	router.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	router.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	router.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	router.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	router.Handle("/debug/pprof/{cmd}", http.HandlerFunc(pprof.Index)) // special handling for Gorilla mux

	err := http.ListenAndServe("localhost:6060", router)
	if err != nil {
		panic(fmt.Sprintf("starting pprof server failed %s", err))
	}
}