package incident20230120

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/client"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/dynamicclients"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoutils"
	"net/http"
	"net/url"
)

const (
	defaultAuthProviderName = "Red Hat SSO"
	authProviderOIDCType    = "oidc"
	authProvidersAPIPath    = "/v1/authProviders"
)

type migrator struct {
	url           string
	adminPassword string
	clientID      string
	clientSecret  string // pragma: allowlist secret
	orgID         string

	client        *client.Client
	dynamicClient *api.AcsTenantsApiService
}

func Command() *cobra.Command {
	m := &migrator{}

	cmd := &cobra.Command{
		Use: "incident-20230120",
		Long: `This migration is based on the incident 20230120, which was reported due to a multitude of issues with the current setup
of the auth providers and groups.

For more information, visit: https://srox.slack.com/archives/C04L0BUNRKN/p1674263826620659
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := m.Construct(); err != nil {
				return errors.Wrap(err, "constructing all dependencies")
			}
			if err := m.Migrate(); err != nil {
				return errors.Wrapf(err, "migrating the instance %s", m.url)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&m.url, "url", "u", "",
		"URL of the central instance which should be migrated. The format should be https://<hostname>:<port> "+
			"and should be the UI URL, not the data URL.")
	cmd.Flags().StringVarP(&m.adminPassword, "password", "p", "",
		"password for the administrative user. This will be used for Basic Auth with central.")
	cmd.Flags().StringVar(&m.clientID, "client-id", "",
		"client ID for a service account authorized to access the sso.r.c dynamic client API")
	cmd.Flags().StringVar(&m.clientSecret, "client-secret", "", // pragma: allowlist secret
		"client secret for a service account authorized to access the sso.r.c dynamic client API")
	cmd.Flags().StringVar(&m.orgID, "org-id", "", "organisation ID associated with the central.")

	cmd.MarkFlagsRequiredTogether("url", "password", "client-id", "client-secret", "org-id")

	return cmd
}

func (m *migrator) Construct() error {
	// Central client currently has a dependency towards the private.ManagedCentral, however it's only used for reading
	// contents for error message.
	m.client = client.NewCentralClient(private.ManagedCentral{
		Metadata: private.ManagedCentralAllOfMetadata{
			Name:      m.url,
			Namespace: "default",
		},
	}, m.url, m.adminPassword)

	// Dynamic client requires a realm config, which contains the client ID / secret + token URLs.
	m.dynamicClient = dynamicclients.NewDynamicClientsAPI(&iam.IAMRealmConfig{
		BaseURL:          "https://sso.redhat.com",
		Realm:            "redhat-external",
		ClientID:         m.clientID,
		ClientSecret:     m.clientSecret, // pragma: allowlist secret
		GrantType:        "client_credentials",
		TokenEndpointURI: "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token",
		APIEndpointURI:   "/auth/realms/redhat-external",
	})

	return nil
}

func (m *migrator) Migrate() error {
	// 1. Retrieve all existing information from the central.
	//    This includes:
	//    - the existing auth provider ID.
	//    - the existing dynamic sso.r.c client ID.
	//    - the existing groups associated with the auth provider.
	existingAuthProviderID, existingClientID, existingGroups, err := m.retrieveAuthProviderRelatedData()
	if err != nil {
		return errors.Wrap(err, "retrieving auth provider related data")
	}

	// 2. Delete the existing auth provider.
	//    Note: we cannot create a secondary auth provider, since the name must be unique across auth providers.
	if err := m.client.SendRequestToCentral(context.Background(), nil, http.MethodDelete,
		fmt.Sprintf("%s/%s", authProvidersAPIPath, existingAuthProviderID), nil); err != nil {
		return errors.Wrapf(err, "attempting to delete auth provider %q", err)
	}

	// 3. Request a new dynamic client.
	//	  Note: we cannot retrieve the client secret of existing clients, hence we need to re-create it.
	newDynamicClient, _, err := m.dynamicClient.CreateAcsClient(context.Background(), api.AcsClientRequestData{
		Name:         "acsms-XXXX", // TODO
		RedirectUris: []string{fmt.Sprintf("https://%s/sso/providers/oidc/callback", m.url)},
		OrgId:        m.orgID,
	})
	if err != nil {
		return errors.Wrapf(err, "creating new dynamic client for central %s", m.url)
	}

	// 4. Create the new auth provider.
	//    The new auth provider will:
	//	  - use the newly created dynamic client (TBD: not sure if this will require users to login once again).
	//    - the organisation ID associated with the central will be a required attribute.
	//    - an additional claim mapping will be created, which maps the account_id claim to orgid.
	newAuthProviderID, err := m.createNewAuthProvider(newDynamicClient.ClientId, newDynamicClient.Secret)
	if err != nil {
		return errors.Wrapf(err, "creating new auth provider for central %s", m.url)
	}

	// 5. Migrate the previously existing groups to the newly created auth provider.
	//    The previously existing group's auth provider ID will be moved to the newly created auth provider's ID.
	if err := m.migrateGroups(existingGroups, newAuthProviderID); err != nil {
		return errors.Wrapf(err, "migrating groups for auth provider %q", newAuthProviderID)
	}

	// 6. Cleanup the previously existing dynamic client.
	if resp, err := m.dynamicClient.DeleteAcsClient(context.Background(), existingClientID); err != nil {
		if resp.StatusCode == http.StatusNotFound {
			glog.V(7).Infof("dynamic client %s could not be found; skipping the deletion", existingClientID)
		} else {
			return errors.Wrapf(err, "deleting dynamic client %q", existingClientID)
		}
	}

	return nil
}

func (m *migrator) retrieveAuthProviderRelatedData() (string, string, []*storage.Group, error) {
	// 1. Retrieve the existing auth provider ID.
	authProviders, err := m.client.GetLoginAuthProviders(context.Background())
	if err != nil {
		return "", "", nil, err
	}
	var authProviderID string
	for _, provider := range authProviders.GetAuthProviders() {
		if provider.GetType() == "oidc" && provider.GetName() == "Red Hat SSO" {
			authProviderID = provider.GetId()
		}
	}
	if authProviderID == "" {
		return "", "", nil, errors.Errorf("no default sso.r.c auth provider found for central %s", m.url)
	}

	// 2. Retrieve the client ID of the existing auth provider.
	var authProvider storage.AuthProvider
	if err := m.client.SendRequestToCentral(context.Background(), nil, http.MethodGet, fmt.Sprintf(
		"%s/%s", authProvidersAPIPath, authProviderID), &authProvider); err != nil {
		return "", "", nil, errors.Wrapf(err, "retrieving exisiting auth provider %q", authProviderID)
	}
	existingClientID, exists := authProvider.GetConfig()["client_id"]
	if !exists {
		return "", "", nil, errors.Errorf("no client_id found for auth provider %q", authProviderID)
	}

	// 3. Retrieve the groups associated with the existing auth provider.
	groups, err := getGroupsByAuthProviderID(authProviderID, m.client)
	if err != nil {
		return "", "", nil, err
	}

	return authProviderID, existingClientID, groups, nil
}

func (m *migrator) createNewAuthProvider(clientID, clientSecret string) (string, error) {
	url, err := url.Parse(m.url)
	if err != nil {
		return "", errors.Wrapf(err, "parsing URL %s", m.url)
	}

	authProviderRequest := &v1.PostAuthProviderRequest{
		Provider: &storage.AuthProvider{
			Name:       defaultAuthProviderName,
			Type:       authProviderOIDCType,
			UiEndpoint: url.Hostname(),
			Enabled:    true,
			Config: map[string]string{
				"client_id":     clientID,
				"client_secret": clientSecret, // pragma: allowlist secret
				"mode":          "post",
			},
			Active: true,
			RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
				{
					AttributeKey:   "orgid",
					AttributeValue: m.orgID,
				},
			},
			Traits: &storage.Traits{MutabilityMode: storage.Traits_ALLOW_MUTATE_FORCED},
			ClaimMappings: map[string]string{
				"account_id": "orgid",
			},
		},
	}

	var newAuthProvider storage.AuthProvider
	if err := m.client.SendRequestToCentral(context.Background(), authProviderRequest, http.MethodPost,
		authProvidersAPIPath, &newAuthProvider); err != nil {
		return "", errors.Wrapf(err, "creating new auth provider for central %s", m.url)
	}

	return newAuthProvider.GetId(), nil
}

func (m *migrator) migrateGroups(groups []*storage.Group, authProviderID string) error {
	// Patch the previous group's auth provider ID to the new auth provider ID.
	for _, group := range groups {
		group.GetProps().AuthProviderId = authProviderID
	}

	groupsBatchRequest := &v1.GroupBatchUpdateRequest{
		PreviousGroups: nil,
		RequiredGroups: groups,
	}
	if err := m.client.SendRequestToCentral(context.Background(), groupsBatchRequest, http.MethodPost,
		"/v1/groupsbatch", nil); err != nil {
		return errors.Wrapf(err, "updating groups to auth provider %q", authProviderID)
	}

	// Verify the groups are as expected.
	updatedGroups, err := getGroupsByAuthProviderID(authProviderID, m.client)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if !protoutils.SliceContains(group, updatedGroups) {
			return errors.Wrapf(err, "group %+v was expected but not found for auth provider %q",
				group, authProviderID)
		}
	}

	return nil
}

func getGroupsByAuthProviderID(authProviderID string, client *client.Client) ([]*storage.Group, error) {
	var groups v1.GetGroupsResponse
	if err := client.SendRequestToCentral(context.Background(),
		&v1.GetGroupsRequest{AuthProviderIdOpt: &v1.GetGroupsRequest_AuthProviderId{AuthProviderId: authProviderID}},
		http.MethodGet, "/v1/groups", &groups); err != nil {
		return nil, errors.Wrapf(err, "retrieving groups associated with auth provider %q",
			authProviderID)
	}

	return groups.GetGroups(), nil
}
