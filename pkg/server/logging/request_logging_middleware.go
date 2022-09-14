package logging

import (
	"context"
	"net/http"
	"time"

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
		// namedRoute := mux.CurrentRoute(request).GetName()
		ctx := request.Context()
		ctx = context.WithValue(ctx, logging.RemoteAddrKey, request.RemoteAddr)
		request = request.WithContext(ctx)

		loggingWriter := NewLoggingWriter(writer, request, logger, NewJSONLogFormatter())
		loggingWriter.prepareReRequestLog(request)
		loggingWriter.Info("Request received")
		before := time.Now()
		handler.ServeHTTP(loggingWriter, request)
		elapsed := time.Since(before).String()
		// 	logger.SugaredLogger = logger.SugaredLogger.With(
		// 		"elapsed", elapsed,
		// 		"status", loggingWriter.responseStatus,
		// )
		loggingWriter.prepareResponseLog(elapsed)
		loggingWriter.Info("Response sent")

		// logEvent := logger.NewLogEventFromString(mux.CurrentRoute(request).GetName())
		// action := logEvent.Type
		// ctx := request.Context()
		// if action != "" {
		// 	ctx = context.WithValue(request.Context(), logger.ActionKey, action)
		// }
		// ctx = context.WithValue(ctx, logger.RemoteAddrKey, request.RemoteAddr)
		// request = request.WithContext(ctx)

		// loggingWriter := NewLoggingWriter(writer, request, NewJSONLogFormatter())
		// loggingWriter.Log(loggingWriter.prepareRequestLog())
		// before := time.Now()
		// handler.ServeHTTP(loggingWriter, request)
		// elapsed := time.Since(before).String()
		// loggingWriter.Log(loggingWriter.prepareResponseLog(elapsed))
	})
}
