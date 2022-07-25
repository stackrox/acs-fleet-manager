package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/golang/glog"
)

// NewJSONLogFormatter ...
func NewJSONLogFormatter() *jsonLogFormatter {
	return &jsonLogFormatter{}
}

type jsonLogFormatter struct{}

var _ LogFormatter = &jsonLogFormatter{}

// FormatRequestLog ...
func (f *jsonLogFormatter) FormatRequestLog(r *http.Request) (string, error) {
	jsonlog := jsonRequestLog{
		Method:     r.Method,
		RequestURI: r.RequestURI,
		RemoteAddr: r.RemoteAddr,
	}
	if glog.V(10) {
		jsonlog.Header = r.Header
		jsonlog.Body = r.Body
	}

	log, err := json.Marshal(jsonlog)
	if err != nil {
		return "", fmt.Errorf("marshalling json: %w", err)
	}
	return string(log[:]), nil
}

// FormatResponseLog ...
func (f *jsonLogFormatter) FormatResponseLog(info *ResponseInfo) (string, error) {
	jsonlog := jsonResponseLog{Header: nil, Status: info.Status, Elapsed: info.Elapsed}
	if glog.V(10) {
		jsonlog.Body = string(info.Body[:])
	}
	log, err := json.Marshal(jsonlog)
	if err != nil {
		return "", fmt.Errorf("marshalling json: %w", err)
	}
	return string(log[:]), nil
}

// FormatObject ...
func (f *jsonLogFormatter) FormatObject(o interface{}) (string, error) {
	log, err := json.Marshal(o)
	if err != nil {
		return "", fmt.Errorf("marshalling json: %w", err)
	}
	return string(log), nil
}

type jsonRequestLog struct {
	Method     string        `json:"request_method"`
	RequestURI string        `json:"request_url"`
	Header     http.Header   `json:"request_header,omitempty"`
	Body       io.ReadCloser `json:"request_body,omitempty"`
	RemoteAddr string        `json:"request_remote_ip,omitempty"`
}

type jsonResponseLog struct {
	Header  http.Header `json:"response_header,omitempty"`
	Status  int         `json:"response_status,omitempty"`
	Body    string      `json:"response_body,omitempty"`
	Elapsed string      `json:"elapsed,omitempty"`
}
