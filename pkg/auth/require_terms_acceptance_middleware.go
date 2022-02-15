package auth

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
)

type RequireTermsAcceptanceMiddleware interface {
	// RequireTermsAcceptance will check that the user has accepted the required terms.
	// The current implementation is backed by OCM and can be disabled with the "enabled" flag set to false.
	RequireTermsAcceptance(enabled bool, amsClient ocm.AMSClient, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
}

type requireTermsAcceptanceMiddleware struct {
	cache *cache.Cache
}

var _ RequireTermsAcceptanceMiddleware = &requireTermsAcceptanceMiddleware{}

func NewRequireTermsAcceptanceMiddleware() RequireTermsAcceptanceMiddleware {
	return &requireTermsAcceptanceMiddleware{
		// entries will expire in 5 minutes and will be evicted every 10 minutes
		cache: cache.New(5*time.Minute, 10*time.Minute),
	}
}

func (m *requireTermsAcceptanceMiddleware) RequireTermsAcceptance(enabled bool, amsClient ocm.AMSClient, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if enabled {
				ctx := request.Context()
				claims, err := GetClaimsFromContext(ctx)
				if err != nil {
					shared.HandleError(request, writer, errors.NewWithCause(code, err, ""))
					return
				}
				username := GetUsernameFromClaims(claims)
				termsRequired, cached := m.cache.Get(username)
				if !cached {
					termsRequired, _, err = amsClient.GetRequiresTermsAcceptance(username)
					if err != nil {
						shared.HandleError(request, writer, errors.NewWithCause(code, err, ""))
						return
					}

					m.cache.Set(username, termsRequired, cache.DefaultExpiration)
				}

				if termsRequired.(bool) {
					shared.HandleError(request, writer, errors.New(code, "required terms have not been accepted"))
					return
				}
			}

			next.ServeHTTP(writer, request)
		})
	}
}
