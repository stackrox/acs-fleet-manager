package logging

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/logging"
)

// RequestLoggingMiddlewareWithLogger ...
func RequestLoggingMiddlewareWithLogger(logger *logging.Logger) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return RequestLoggingMiddleware(logger, handler)
	}
}

// RequestLoggingMiddleware ...
func RequestLoggingMiddleware(logger *logging.Logger, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		namedRoute := mux.CurrentRoute(request).GetName()
		ctx := request.Context()
		ctx = context.WithValue(ctx, logging.RemoteAddrKey, request.RemoteAddr)
		request = request.WithContext(ctx)

		loggingWriter := NewLoggingWriter(writer, request, logger, NewJSONLogFormatter())
		loggingWriter.Info("Request received")
		before := time.Now()
		handler.ServeHTTP(loggingWriter, request)
		elapsed := time.Since(before).String()
		loggingWriter.Infow("Response sent",
			"route", namedRoute,
			"method", request.Method,
			"elapsed", elapsed,
			"status", loggingWriter.GetResponseStatusCode(),
		)
	})
}
