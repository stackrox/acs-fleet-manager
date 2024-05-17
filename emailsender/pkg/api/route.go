package api

import (
	"github.com/gorilla/mux"
	"net/http"
)

func SetRoutes(router *mux.Router) {
	router.HandleFunc("/test/{id}", func(rw http.ResponseWriter, req *http.Request) {})
}
