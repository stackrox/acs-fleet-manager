package handlers

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services/quota"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
)

type cloudAccountsHandler struct {
	client ocm.AMSClient
}

// NewCloudAccountsHandler ...
func NewCloudAccountsHandler(client ocm.AMSClient) *cloudAccountsHandler {
	return &cloudAccountsHandler{
		client: client,
	}
}

// Get ...
func (h *cloudAccountsHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: h.actionFunc(r),
	}
	handlers.HandleGet(w, r, cfg)
}

func (h *cloudAccountsHandler) actionFunc(r *http.Request) func() (i interface{}, serviceError *errors.ServiceError) {
	return func() (i interface{}, serviceError *errors.ServiceError) {
		ctx := r.Context()
		claims, err := auth.GetClaimsFromContext(ctx)
		if err != nil {
			return nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
		}
		orgID, err := claims.GetOrgID()
		if err != nil {
			return nil, errors.NewWithCause(errors.ErrorForbidden, err, "cannot make request without orgID claim")
		}
		organizationID, err := h.client.GetOrganisationIDFromExternalID(orgID)
		if err != nil {
			return nil, errors.OrganisationNotFound(orgID, err)
		}

		cloudAccounts, err := h.client.GetCustomerCloudAccounts(organizationID, []string{quota.RHACSMarketplaceQuotaID})
		if err != nil {
			return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to fetch cloud accounts from AMS")
		}
		return presenters.PresentCloudAccounts(cloudAccounts), nil
	}
}
