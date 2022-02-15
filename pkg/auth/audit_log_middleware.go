package auth

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/server/logging"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
	"io"
	"net/http"
)

type AuditLogMiddleware interface {
	AuditLog(code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
}

type auditInfo struct {
	Type               string        `json:"type"`
	Username           string        `json:"username"`
	Method             string        `json:"request_method,omitempty"`
	RequestURI         string        `json:"request_url,omitempty"`
	Body               io.ReadCloser `json:"request_body,omitempty"`
	RemoteAddr         string        `json:"request_remote_ip,omitempty"`
	ResponseStatusCode int           `json:"response_status_code,omitempty"`
}

type auditLogMiddleware struct {
}

var _ AuditLogMiddleware = &auditLogMiddleware{}

func NewAuditLogMiddleware() AuditLogMiddleware {
	return &auditLogMiddleware{}
}

func (a *auditLogMiddleware) AuditLog(code errors.ServiceErrorCode) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			claims, err := GetClaimsFromContext(ctx)
			serviceErr := errors.New(code, "")
			if err != nil {
				shared.HandleError(request, writer, serviceErr)
				return
			}
			username := GetUsernameFromClaims(claims)
			info := auditInfo{
				Type:       "audit",
				Username:   username,
				Method:     request.Method,
				RequestURI: request.RequestURI,
				Body:       request.Body,
				RemoteAddr: request.RemoteAddr,
			}
			logWriter := logging.NewLoggingWriter(writer, request, logging.NewJSONLogFormatter())
			err = logWriter.LogObject(info, nil)
			if err != nil {
				shared.HandleError(request, writer, serviceErr)
				return
			}
			next.ServeHTTP(logWriter, request)
			statusCode := logWriter.GetResponseStatusCode()
			info = auditInfo{
				Type:               "audit",
				ResponseStatusCode: statusCode,
			}
			// the logger will also add the operationId prefix to each of the log message, which can then be used to associated the two log messages together
			err = logWriter.LogObject(info, nil)
			if err != nil {
				// response is already returned, just log the error if there is any
				logWriter.Log(fmt.Sprintf("failed to log object %v", info), err)
				return
			}
		})
	}
}
