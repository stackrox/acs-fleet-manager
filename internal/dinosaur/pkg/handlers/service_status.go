package handlers

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

type serviceStatusHandler struct {
	dinosaurService   services.DinosaurService
	accessControlList *acl.AccessControlListConfig
}

// NewServiceStatusHandler ...
func NewServiceStatusHandler(service services.DinosaurService, accessControlList *acl.AccessControlListConfig) *serviceStatusHandler {
	return &serviceStatusHandler{
		dinosaurService:   service,
		accessControlList: accessControlList,
	}
}

// Get ...
func (h serviceStatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			context := r.Context()
			claims, err := auth.GetClaimsFromContext(context)
			if err != nil {
				return presenters.PresentServiceStatus(), nil
			}

			username, _ := claims.GetUsername()
			accessControlListConfig := h.accessControlList
			if accessControlListConfig.EnableDenyList {
				userIsDenied := accessControlListConfig.DenyList.IsUserDenied(username)
				if userIsDenied {
					glog.V(5).Infof("User %s is denied to access the service", username)
					return presenters.PresentServiceStatus(), nil
				}
			}

			return presenters.PresentServiceStatus(), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}
