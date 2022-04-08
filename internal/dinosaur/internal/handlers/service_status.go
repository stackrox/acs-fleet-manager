package handlers

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/services"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"

	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/golang/glog"
)

type serviceStatusHandler struct {
	dinosaurService   services.DinosaurService
	accessControlList *acl.AccessControlListConfig
}

func NewServiceStatusHandler(service services.DinosaurService, accessControlList *acl.AccessControlListConfig) *serviceStatusHandler {
	return &serviceStatusHandler{
		dinosaurService:   service,
		accessControlList: accessControlList,
	}
}

func (h serviceStatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			context := r.Context()
			claims, err := auth.GetClaimsFromContext(context)
			if err != nil {
				return presenters.PresentServiceStatus(true, false), nil
			}

			username := auth.GetUsernameFromClaims(claims)
			accessControlListConfig := h.accessControlList
			if accessControlListConfig.EnableDenyList {
				userIsDenied := accessControlListConfig.DenyList.IsUserDenied(username)
				if userIsDenied {
					glog.V(5).Infof("User %s is denied to access the service. Setting dinosaur maximum capacity to 'true'", username)
					return presenters.PresentServiceStatus(true, false), nil
				}
			}

			hasAvailableDinosaurCapacity, capacityErr := h.dinosaurService.HasAvailableCapacity()
			return presenters.PresentServiceStatus(false, !hasAvailableDinosaurCapacity), capacityErr
		},
	}
	handlers.HandleGet(w, r, cfg)
}
