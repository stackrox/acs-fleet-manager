package auth

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// RequireIssuerMiddleware ...
type RequireIssuerMiddleware interface {
	// RequireIssuer checks if the iss field in the JWT claim matches one of the given issuers.
	// If it does not, then the specified code is returned.
	RequireIssuer(issuers []string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
}

type requireIssuerMiddleware struct {
}

var _ RequireIssuerMiddleware = &requireIssuerMiddleware{}

// NewRequireIssuerMiddleware ...
func NewRequireIssuerMiddleware() RequireIssuerMiddleware {
	return &requireIssuerMiddleware{}
}

// RequireIssuer ...
func (m *requireIssuerMiddleware) RequireIssuer(issuers []string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			claims, err := GetClaimsFromContext(ctx)
			serviceErr := errors.New(code, "")
			if err != nil {
				shared.HandleError(request, writer, serviceErr)
				return
			}

			issuerAccepted := false
			for _, issuer := range issuers {
				if claims.VerifyIssuer(issuer, true) {
					issuerAccepted = true
					break
				}
			}

			if !issuerAccepted {
				shared.HandleError(request, writer, serviceErr)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}
