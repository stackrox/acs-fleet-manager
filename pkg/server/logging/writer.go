package logging

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
)

func redactRequest(request *http.Request) *http.Request {
	secretNames := []string{"authorization"}
	redacted := "REDACTED"
	requestCopy := *request

	requestCopy.Header = make(map[string][]string, len(request.Header))
NEXT_HEADER:
	for headerName, headerValue := range request.Header {
		for _, secretName := range secretNames {
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
func NewLoggingWriter(w http.ResponseWriter, r *http.Request, f LogFormatter) *loggingWriter {
	r = redactRequest(r)
	return &loggingWriter{ResponseWriter: w, request: r, formatter: f}
}

type loggingWriter struct {
	http.ResponseWriter
	request        *http.Request
	formatter      LogFormatter
	responseStatus int
	responseBody   []byte
}

// Flush ...
func (writer *loggingWriter) Flush() {
	writer.ResponseWriter.(http.Flusher).Flush()
}

// Write ...
func (writer *loggingWriter) Write(body []byte) (int, error) {
	writer.responseBody = body
	return writer.ResponseWriter.Write(body)
}

// WriteHeader ...
func (writer *loggingWriter) WriteHeader(status int) {
	writer.responseStatus = status
	writer.ResponseWriter.WriteHeader(status)
}

// Log ...
func (writer *loggingWriter) Log(log string, err error) {
	ulog := logger.NewUHCLogger(writer.request.Context())
	switch err {
	case nil:
		ulog.V(LoggingThreshold).Infof(log)
	default:
		ulog.Error(errors.Wrap(err, "Unable to format request/response for log."))
	}
}

// LogObject ...
func (writer *loggingWriter) LogObject(o interface{}, err error) error {
	log, merr := writer.formatter.FormatObject(o)
	if merr != nil {
		return merr
	}
	writer.Log(log, err)
	return nil
}

// GetResponseStatusCode ...
func (writer *loggingWriter) GetResponseStatusCode() int {
	return writer.responseStatus
}

func (writer *loggingWriter) prepareRequestLog() (string, error) {
	return writer.formatter.FormatRequestLog(writer.request)
}

func (writer *loggingWriter) prepareResponseLog(elapsed string) (string, error) {
	info := &ResponseInfo{
		Header:  writer.ResponseWriter.Header(),
		Body:    writer.responseBody,
		Status:  writer.responseStatus,
		Elapsed: elapsed,
	}

	return writer.formatter.FormatResponseLog(info)
}
