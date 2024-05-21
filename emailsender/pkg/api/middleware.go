package api

import (
	"mime"
	"net/http"

	"github.com/golang/glog"
)

// LogRequest logs HTTP request
func LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		glog.Infof("%s %s", r.Method, r.URL)

		next.ServeHTTP(w, r)
	})
}

// EnsureJSONContentType enforces Content-Type: application/json header
func EnsureJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")

		if contentType == "" {
			http.Error(w, "Empty Content-Type", http.StatusBadRequest)
			return
		}
		if contentType != "" {
			mt, _, err := mime.ParseMediaType(contentType)
			if err != nil {
				http.Error(w, "Malformed Content-Type header", http.StatusBadRequest)
				return
			}

			if mt != "application/json" {
				http.Error(w, "Content-Type header must be application/json", http.StatusUnsupportedMediaType)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
