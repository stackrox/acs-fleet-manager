package incident20230120

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/client"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/utils"
	"net/http"
	"net/url"
)

const (
	defaultAuthProviderName = "Red Hat SSO"
	authProviderOIDCType    = "oidc"
	authProvidersAPIPath    = "/v1/authProviders"
)

type migrator struct {
	url             string
	adminPassword   string
	clientID        string
	clientSecret    string // pragma: allowlist secret
	orgID           string
	name            string
	id              string
	migrateOrgAdmin bool

	client        *client.Client
	dynamicClient *api.AcsTenantsApiService
}

func Command() *cobra.Command {
	m := &migrator{}

	cmd := &cobra.Command{
		Use:   "incident-20230120",
		Short: "Migration required for the incident 20230120.",
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
	cmd.Flags().StringVar(&m.clientID, "client-id", "",
		"client ID for the dynamic client associated with central.")
	cmd.Flags().StringVar(&m.clientSecret, "client-secret", "", // pragma: allowlist secret
		"client secret for the dynamic client associated with central.")
	cmd.Flags().StringVar(&m.orgID, "org-id", "", "organisation ID associated with the central.")
	cmd.Flags().StringVar(&m.name, "name", "", "name of the central instance")
	cmd.Flags().StringVar(&m.id, "id", "", "id of the central instance")
	cmd.Flags().BoolVar(&m.migrateOrgAdmin, "migrate-org-admin", false,
		"include the migration of groups that use org_admin to use admin:org:all instead")

	utils.Must(cmd.MarkFlagRequired("url"))
	utils.Must(cmd.MarkFlagRequired("client-id"))
	utils.Must(cmd.MarkFlagRequired("client-secret"))
	utils.Must(cmd.MarkFlagRequired("org-id"))
	utils.Must(cmd.MarkFlagRequired("id"))
	utils.Must(cmd.MarkFlagRequired("name"))

	return cmd
}

func (m *migrator) Construct() error {
	// Enable the admin password for the specific central instance.
	adminPassword, err := shared.EnableAdminPassword(context.Background(), m.id, m.name, m.url)
	if err != nil {
		return errors.Wrapf(err, "enabling admin password for central %s", m.name)
	}
	m.adminPassword = adminPassword // pragma: allowlist secret

	// Central client currently has a dependency towards the private.ManagedCentral, however it's only used for reading
	// contents for error message.
	m.client = client.NewCentralClient(private.ManagedCentral{
		Metadata: private.ManagedCentralAllOfMetadata{
			Name:      m.url,
			Namespace: "default",
		},
	}, m.url, m.adminPassword)

	return nil
}

func (m *migrator) Migrate() error {
	// Ensure we always disable the admin password after migration.
	defer utils.Must(shared.DisableAdminPassword(context.Background(), m.id, m.name))

	url, err := url.Parse(m.url)
	if err != nil {
		return errors.Wrapf(err, "parsing URL %s", m.url)
	}

	// 1. Retrieve all existing information from the central.
	//    This includes:
	//    - the existing auth provider ID.
	//    - the existing dynamic sso.r.c client ID.
	//    - the existing groups associated with the auth provider.
	existingAuthProviderID, existingGroups, err := m.retrieveAuthProviderRelatedData()
	if err != nil {
		return errors.Wrap(err, "retrieving auth provider related data")
	}

	// 2. Delete the existing auth provider.
	//    Note: we cannot create a secondary auth provider, since the name must be unique across auth providers.

	glog.Infof("Sending request to delete auth provider %s for central %s", existingAuthProviderID, m.name)
	if err := m.client.SendRequestToCentral(context.Background(), nil, http.MethodDelete,
		fmt.Sprintf("%s/%s?force", authProvidersAPIPath, existingAuthProviderID), nil); err != nil {
		return errors.Wrapf(err, "attempting to delete auth provider %q", err)
	}

	// 3. Create the new auth provider.
	//    The new auth provider will:
	//	  - use dynamic client.
	//    - the organisation ID associated with the central will be a required attribute.
	//    - an additional claim mapping will be created, which maps the account_id claim to orgid.
	newAuthProviderID, err := m.createNewAuthProvider(url.Hostname())
	if err != nil {
		return errors.Wrapf(err, "creating new auth provider for central %s", m.url)
	}

	// 4. Migrate the previously existing groups to the newly created auth provider.
	//    The previously existing group's auth provider ID will be moved to the newly created auth provider's ID.
	if err := m.migrateGroups(existingGroups, newAuthProviderID); err != nil {
		return errors.Wrapf(err, "migrating groups for auth provider %q", newAuthProviderID)
	}

	return nil
}

func (m *migrator) retrieveAuthProviderRelatedData() (string, []*storage.Group, error) {
	// 1. Retrieve the existing auth provider ID.
	authProviders, err := m.client.GetLoginAuthProviders(context.Background())
	if err != nil {
		return "", nil, errors.Wrapf(err, "retrieving login authproviders for central %s", m.url)
	}

	glog.Infof("Login auth providers found for central %s:\n%+v\n", m.name, authProviders)

	var authProviderID string
	for _, provider := range authProviders.GetAuthProviders() {
		if provider.GetType() == "oidc" && provider.GetName() == "Red Hat SSO" {
			glog.Infof("Found default sso.r.c. auth provider for central %s:\n %+v\n", m.name, provider)
			authProviderID = provider.GetId()
		}
	}
	if authProviderID == "" {
		return "", nil, errors.Errorf("no default sso.r.c auth provider found for central %s", m.url)
	}

	// 2. Retrieve the groups associated with the existing auth provider.
	groups, err := getGroupsByAuthProviderID(authProviderID, m.client)
	if err != nil {
		return "", nil, err
	}

	glog.Infof("Groups associated with auth provider %s for central %s:\n%+v\n", authProviderID, m.name, groups)

	return authProviderID, groups, nil
}

func (m *migrator) createNewAuthProvider(uiEndpoint string) (string, error) {
	authProviderRequest := &v1.PostAuthProviderRequest{
		Provider: &storage.AuthProvider{
			Name:       defaultAuthProviderName,
			Type:       authProviderOIDCType,
			UiEndpoint: uiEndpoint,
			Enabled:    true,
			Config: map[string]string{
				"client_id":     m.clientID,
				"client_secret": m.clientSecret, // pragma: allowlist secret
				"mode":          "post",
			},
			Active: true,
			RequiredAttributes: []*storage.AuthProvider_RequiredAttribute{
				{
					AttributeKey:   "rh_org_id",
					AttributeValue: m.orgID,
				},
			},
			Traits: &storage.Traits{MutabilityMode: storage.Traits_ALLOW_MUTATE_FORCED},
			ClaimMappings: map[string]string{
				"org_id":             "rh_org_id",
				"realm_access.roles": "groups",
			},
		},
	}

	glog.Infof("Send new auth provider request for central %s:\n%+v\n", m.name, authProviderRequest)

	var newAuthProvider storage.AuthProvider
	if err := m.client.SendRequestToCentral(context.Background(), authProviderRequest, http.MethodPost,
		authProvidersAPIPath, &newAuthProvider); err != nil {
		return "", errors.Wrapf(err, "creating new auth provider for central %s", m.url)
	}

	glog.Infof("Newly created auth provider for central %s:\n%+v\n", m.name, newAuthProvider)

	return newAuthProvider.GetId(), nil
}

func (m *migrator) migrateGroups(groups []*storage.Group, authProviderID string) error {

	glog.Infof("Groups before adjusting the auth provider ID for central %s:\n%+v\n", groups)

	// Patch the previous group's auth provider ID to the new auth provider ID.
	for _, group := range groups {
		group.GetProps().AuthProviderId = authProviderID

		if m.migrateOrgAdmin && group.GetProps().GetKey() == "groups" && group.GetProps().GetValue() == "org_admin" {
			group.Props.Value = "admin:org:all"
		}
	}

	glog.Infof("Groups after adjusting the auth provider ID to %s for central %s:\n%+v\n",
		authProviderID, m.name, groups)

	groupsBatchRequest := &v1.GroupBatchUpdateRequest{
		PreviousGroups: nil,
		RequiredGroups: groups,
	}

	glog.Infof("Sending groupsbatch request to central %s:\n%+v\n", m.name, groupsBatchRequest)
	if err := m.client.SendRequestToCentral(context.Background(), groupsBatchRequest, http.MethodPost,
		"/v1/groupsbatch", nil); err != nil {
		return errors.Wrapf(err, "updating groups to auth provider %q", authProviderID)
	}

	// Verify the groups are as expected.
	updatedGroups, err := getGroupsByAuthProviderID(authProviderID, m.client)
	if err != nil {
		return err
	}

	glog.Infof("Groups after executing the groupsbatch request for central %s:\n%+v\n", m.name, updatedGroups)

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
