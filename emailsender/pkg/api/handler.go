package api

import (
	"github.com/gorilla/mux"
	"net/http"
)

func NewHandler() http.Handler {
	r := mux.NewRouter()
}
