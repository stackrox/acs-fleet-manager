package rhsso

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	oidcProviderCallbackPath    = "/sso/providers/oidc/callback"
	dynamicClientsNameMaxLength = 50
)

func AugmentWithDynamicAuthConfig(ctx context.Context, r *dbapi.CentralRequest, realmConfig *iam.IAMRealmConfig, apiClient *api.AcsTenantsApiService) error {
	// There is a limit on name length of the dynamic client. To avoid unnecessary errors,
	// we truncate name here.
	name := stringutils.Truncate(fmt.Sprintf("acscs-%s", r.Name), dynamicClientsNameMaxLength)
	orgID := r.OrganisationID
	redirectURIs := []string{fmt.Sprintf("https://%s%s", r.GetUIHost(), oidcProviderCallbackPath)}

	dynamicClientData, _, err := apiClient.CreateAcsClient(ctx, api.AcsClientRequestData{
		Name:         name,
		OrgId:        orgID,
		RedirectUris: redirectURIs,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create RHSSO dynamic client for %s", r.ID)
	}

	r.AuthConfig.ClientID = dynamicClientData.ClientId
	r.AuthConfig.ClientSecret = dynamicClientData.Secret // pragma: allowlist secret
	r.AuthConfig.Issuer = realmConfig.ValidIssuerURI
	return nil
}
