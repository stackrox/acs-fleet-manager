package auth

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

type RequireOrgIDMiddleware interface {
	// RequireOrgID will check that org_id is set as part of the JWT claims in the
	// request and that it is not empty and return code ServiceErrorCode in case
	// the previous conditions are not true
	RequireOrgID(code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
}

type requireOrgIDMiddleware struct {
}

var _ RequireOrgIDMiddleware = &requireOrgIDMiddleware{}

func NewRequireOrgIDMiddleware() RequireOrgIDMiddleware {
	return &requireOrgIDMiddleware{}
}

func (m *requireOrgIDMiddleware) RequireOrgID(code errors.ServiceErrorCode) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			claims, err := GetClaimsFromContext(ctx)
			serviceErr := errors.New(code, "")
			if err != nil {
				shared.HandleError(request, writer, serviceErr)
				return
			}

			orgID, _ := claims.GetOrgId()
			if orgID == "" {
				shared.HandleError(request, writer, serviceErr)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}
