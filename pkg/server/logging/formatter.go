package logging

import "net/http"

// LogFormatter ...
type LogFormatter interface {
	FormatRequestLog(request *http.Request) (string, error)
	FormatResponseLog(responseInfo *ResponseInfo) (string, error)
	FormatObject(o interface{}) (string, error)
}
