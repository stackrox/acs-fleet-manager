package logging

import (
	"fmt"
	"net/http"
	"strings"

	logger "github.com/stackrox/acs-fleet-manager/pkg/logging"
)

func redactRequest(request *http.Request) *http.Request {
	secretNames := []string{"authorization"}
	redacted := "REDACTED"
	requestCopy := *request

	requestCopy.Header = make(map[string][]string, len(request.Header))
NEXT_HEADER:
	for headerName, headerValue := range request.Header {
		for _, secretName := range secretNames { // pragma: allowlist secret
			if strings.EqualFold(headerName, secretName) {
				// Redact this header.
				requestCopy.Header[headerName] = []string{redacted}
				continue NEXT_HEADER
			}
		}
		requestCopy.Header[headerName] = headerValue
	}
	return &requestCopy
}

// NewLoggingWriter ...
func NewLoggingWriter(w http.ResponseWriter, r *http.Request, logger *logger.Logger, f LogFormatter) *loggingWriter {
	r = redactRequest(r)
	return &loggingWriter{ResponseWriter: w, request: r, Logger: logger, formatter: f}
}

type loggingWriter struct {
	http.ResponseWriter
	request        *http.Request
	responseStatus int
	responseBody   []byte
	*logger.Logger
	formatter LogFormatter
}

// Flush ...
func (writer *loggingWriter) Flush() {
	writer.ResponseWriter.(http.Flusher).Flush()
}

// Write ...
func (writer *loggingWriter) Write(body []byte) (int, error) {
	writer.responseBody = body
	i, err := writer.ResponseWriter.Write(body)
	if err != nil {
		return i, fmt.Errorf("writing body: %w", err)
	}
	return i, nil
}

// WriteHeader ...
func (writer *loggingWriter) WriteHeader(status int) {
	writer.responseStatus = status
	writer.ResponseWriter.WriteHeader(status)
}

// LogObject ...
func (writer *loggingWriter) LogObject(o interface{}, err error) error {
	log, merr := writer.formatter.FormatObject(o)
	if merr != nil {
		return fmt.Errorf("formatting object: %w", merr)
	}
	writer.Logger.Infof("%s: %v", log, err)
	return nil
}

// GetResponseStatusCode ...
func (writer *loggingWriter) GetResponseStatusCode() int {
	return writer.responseStatus
}

func (writer *loggingWriter) prepareReRequestLog(request *http.Request) {
	// TODO(mclasmei)
}

func (writer *loggingWriter) prepareResponseLog(elapsed string) {
	writer.Logger.SugaredLogger = writer.Logger.SugaredLogger.With(
		"status", writer.responseStatus,
		"elapsed", elapsed,
	)

	// info := &ResponseInfo{
	// 	Header:  writer.ResponseWriter.Header(),
	// 	Body:    writer.responseBody,
	// 	Status:  writer.responseStatus,
	// 	Elapsed: elapsed,
	// }

	// s, err := writer.formatter.FormatResponseLog(info)
	// if err != nil {
	// 	return s, fmt.Errorf("formatting request: %w", err)
	// }
	// return s, nil
}
