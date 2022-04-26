package handlers

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/services"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"

	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/golang/glog"
)

type serviceStatusHandler struct {
	centralService   services.CentralService
	accessControlList *acl.AccessControlListConfig
}

func NewServiceStatusHandler(service services.CentralService, accessControlList *acl.AccessControlListConfig) *serviceStatusHandler {
	return &serviceStatusHandler{
		centralService:   service,
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

			hasAvailableCapacity, capacityErr := h.centralService.HasAvailableCapacity()
			return presenters.PresentServiceStatus(false, !hasAvailableCapacity), capacityErr
		},
	}
	handlers.HandleGet(w, r, cfg)
}
