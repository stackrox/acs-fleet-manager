package api

import (
	"mime"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/emailsender/config"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

const ocmIssuer = "https://sso.redhat.com/auth/realms/redhat-external"
const centralServiceAccountRegEx = "system:serviceaccount:rhacs-[a-z0-9]*:central"

// EnsureJSONContentType enforces Content-Type: application/json header
func EnsureJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")

		if contentType == "" {
			http.Error(w, "Empty Content-Type", http.StatusBadRequest)
			return
		}
		if contentType != "" {
			mt, _, err := mime.ParseMediaType(contentType)
			if err != nil {
				http.Error(w, "Malformed Content-Type header", http.StatusBadRequest)
				return
			}

			if mt != "application/json" {
				http.Error(w, "Content-Type header must be application/json", http.StatusUnsupportedMediaType)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func emailsenderAuthorizationMiddleware(authConfig config.AuthConfig) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			claims, err := auth.GetClaimsFromContext(ctx)
			if err != nil {
				shared.HandleError(req, w, errors.Unauthorized("invalid token claims"))
				return
			}

			if claims.VerifyIssuer(ocmIssuer, true) {
				// only check for org ID if we're using OCM tokens
				next = auth.CheckAllowedOrgIDs(authConfig.AllowedOrgIDs)(next)
				next = auth.NewRequireOrgIDMiddleware().RequireOrgID(errors.ErrorUnauthorized)(next)
			} else {
				// in this case we expect a k8s service account token
				// so we need to check for the sub
				next = checkCentralServiceAccountSubject()(next)
			}

			next = auth.CheckAudience(authConfig.AllowedAudiences)(next)
			next = auth.NewRequireIssuerMiddleware().RequireIssuer(authConfig.AllowedIssuer, errors.ErrorUnauthorized)(next)

			next.ServeHTTP(w, req)
		})
	}
}

func checkCentralServiceAccountSubject() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			claims, err := auth.GetClaimsFromContext(ctx)
			if err != nil {
				shared.HandleError(req, w, errors.Unauthorized("invalid token claims"))
				return
			}

			sub, err := claims.GetSubject()
			if err != nil {
				shared.HandleError(req, w, errors.Unauthorized("failed to get subject from token claims"))
				return
			}

			match, err := regexp.MatchString(centralServiceAccountRegEx, sub)
			if err != nil || !match {
				shared.HandleError(req, w, errors.Unauthorized("failed to match subject"))
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}
