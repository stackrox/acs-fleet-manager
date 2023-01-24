// Package incident20230120 provides everything to resolve the incident 20230120.
package incident20230120

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/client"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
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
	name          string
	id            string
	dir           string

	centralClient *client.Client
}

// Command provides the command and all flags to run the migration of the incident 20230120.
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
		SilenceUsage: true,
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
	cmd.Flags().StringVar(&m.dir, "output-directory", "", "output directory where the access logs"+
		" should be stored.")

	utils.Must(cmd.MarkFlagRequired("url"))
	utils.Must(cmd.MarkFlagRequired("client-id"))
	utils.Must(cmd.MarkFlagRequired("client-secret"))
	utils.Must(cmd.MarkFlagRequired("org-id"))
	utils.Must(cmd.MarkFlagRequired("id"))
	utils.Must(cmd.MarkFlagRequired("name"))

	return cmd
}

func (m *migrator) Construct() error {
	// If no output directory is given, use the current working directory.
	if m.dir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "retrieving working directory")
		}
		m.dir = dir
	}

	// Enable the admin password for the specific central instance. This will also ensure that the basic auth provider
	// is ready to use and accepts the password.
	adminPassword, err := shared.EnableAdminPassword(context.Background(), m.id, m.name, m.url)
	if err != nil {
		return errors.Wrapf(err, "enabling admin password and basic auth provider for central %s", m.name)
	}
	m.adminPassword = adminPassword // pragma: allowlist secret

	// Central client currently has a dependency towards the private.ManagedCentral, however it's only used for reading
	// metadata for error message, and ideally shouldn't even have this dependency.
	m.centralClient = client.NewCentralClient(private.ManagedCentral{
		Metadata: private.ManagedCentralAllOfMetadata{
			Name:      m.url,
			Namespace: "default",
		},
	}, m.url, m.adminPassword)

	return nil
}

func (m *migrator) Migrate() error {
	// Ensure we always disable the admin password.
	defer utils.Must(shared.DisableAdminPassword(context.Background(), m.id, m.name))

	url, err := url.Parse(m.url)
	if err != nil {
		return errors.Wrapf(err, "parsing URL %s", m.url)
	}

	// 1. Retrieve all existing information from the central.
	//    This includes:
	//    - the existing auth provider ID.
	//    - the existing groups associated with the auth provider.
	existingAuthProviderID, existingGroups, err := m.retrieveAuthProviderAndGroups()
	if err != nil {
		return errors.Wrap(err, "retrieving auth provider related data")
	}

	// 2. Delete the existing auth provider.
	//    Note: we cannot create a secondary auth provider, since the name must be unique across auth providers.
	glog.Infof("Sending request to delete auth provider %s for central %s", existingAuthProviderID, m.name)
	if err := m.centralClient.SendRequestToCentral(context.Background(), nil, http.MethodDelete,
		fmt.Sprintf("%s/%s?force=true", authProvidersAPIPath, existingAuthProviderID), nil); err != nil {
		return errors.Wrapf(err, "attempting to delete auth provider %q", err)
	}
	glog.Infof("Successfully deleted auth provider %s for central %s", existingAuthProviderID, m.name)

	// 3. Create the new auth provider.
	//    The new auth provider will:
	//	  - use the dynamic client ID / secret of the previous auth provider.
	//    - an additional claim mapping will be created, which maps the org_id claim to rh_org_id.
	//    - the organisation ID associated with the central will be a required attribute for the rh_org_id claim.
	newAuthProviderID, err := m.createNewAuthProvider(url.Hostname())
	if err != nil {
		return errors.Wrapf(err, "creating new auth provider for central %s", m.url)
	}

	// 4. Re-create the previously existing groups with references to the newly created auth provider.
	if err := m.migrateGroups(existingGroups, newAuthProviderID); err != nil {
		return errors.Wrapf(err, "migrating groups for auth provider %q", newAuthProviderID)
	}

	// 5. Retrieve the list of users which accessed central.
	//    This will include their ID, attributes, as well as roles with which they were authenticated and authorized.
	if err := m.storeAuthenticatedUsers(); err != nil {
		return errors.Wrapf(err, "storing authenticated users for central %s", m.name)
	}

	return nil
}

func (m *migrator) retrieveAuthProviderAndGroups() (string, []*storage.Group, error) {
	// 1. Retrieve the existing auth provider ID.
	authProviders, err := m.centralClient.GetLoginAuthProviders(context.Background())
	if err != nil {
		return "", nil, errors.Wrapf(err, "retrieving login authproviders for central %s", m.url)
	}

	glog.Infof("Login auth providers found for central %s:\n%+v\n", m.name, authProviders)

	var authProviderID string
	for _, provider := range authProviders.GetAuthProviders() {
		if provider.GetType() == "oidc" && provider.GetName() == "Red Hat SSO" {
			glog.Infof("Found default sso.r.c. auth provider for central %s:\n %+v\n", m.name, provider)
			authProviderID = provider.GetId()
			break
		}
	}
	if authProviderID == "" {
		return "", nil, errors.Errorf("no default sso.r.c auth provider found for central %s", m.url)
	}

	// 2. Retrieve the groups associated with the existing auth provider.
	groups, err := getGroupsByAuthProviderID(authProviderID, m.centralClient)
	if err != nil {
		return "", nil, err
	}

	glog.Infof("Groups associated with auth provider %s for central %s:\n%+v\n", authProviderID, m.name, groups)

	return authProviderID, groups, nil
}

func (m *migrator) createNewAuthProvider(uiEndpoint string) (string, error) {
	authProviderRequest := &storage.AuthProvider{
		Name:       defaultAuthProviderName,
		Type:       authProviderOIDCType,
		UiEndpoint: uiEndpoint,
		Enabled:    true,
		Config: map[string]string{
			"client_id":     m.clientID,
			"client_secret": m.clientSecret, // pragma: allowlist secret
			// Issuer will be same across all environments, as we use prod sso.r.c.
			"issuer":                       "https://sso.redhat.com/auth/realms/redhat-external",
			"mode":                         "post",
			"disable_offline_access_scope": "true",
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
			"is_org_admin":       "rh_is_org_admin",
			"realm_access.roles": "groups",
		},
	}

	glog.Infof("Send new auth provider request for central %s:\n%+v\n", m.name, authProviderRequest)
	newAuthProvider, err := m.centralClient.SendAuthProviderRequest(context.Background(), authProviderRequest)
	if err != nil {
		return "", errors.Wrapf(err, "creating new auth provider for central %s", m.url)
	}

	glog.Infof("Newly created auth provider for central %s:\n%+v\n", m.name, newAuthProvider)

	return newAuthProvider.GetId(), nil
}

func (m *migrator) migrateGroups(groups []*storage.Group, authProviderID string) error {
	glog.Infof("Groups before adjusting the auth provider ID for central %s:\n%+v\n", m.name, groups)
	for _, group := range groups {
		if group.GetProps().GetKey() == "groups" {
			// Explicitly drop the groups mapping to Admin using org_admin, rh_is_org_admin, admin:org:all.
			if group.GetProps().GetValue() == "org_admin" || group.GetProps().GetValue() == "rh_is_org_admin" || group.GetProps().GetValue() == "admin:org:all" {
				continue
			}
		}
		// Patch the previously existing group, this includes:
		// - resetting the ID field, as ID field is not allowed to be set when creating new groups.
		// - setting the auth provider ID to the newly created auth provider ID.
		group.Props.AuthProviderId = authProviderID
		group.Props.Id = ""

		glog.Infof("Sending group request %+v\n", group)
		if err := m.centralClient.SendGroupRequest(context.Background(), group); err != nil {
			return errors.Wrapf(err, "creating group %+v for auth provider %s", group, authProviderID)
		}
		glog.Infof("Successfully created group %+v\n", group)
	}

	// Ensure we have two groups mapping to admin with the org_admin claim and the admin:org:all claim.
	// Note that for the time being, the admin:org:all claim may not work due to limitations on sso.r.c.
	adminGroups := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				Traits:         &storage.Traits{MutabilityMode: storage.Traits_ALLOW_MUTATE_FORCED},
				AuthProviderId: authProviderID,
				Key:            "groups",
				Value:          "rh_is_org_admin",
			},
			RoleName: "Admin",
		},
		{
			Props: &storage.GroupProperties{
				Traits:         &storage.Traits{MutabilityMode: storage.Traits_ALLOW_MUTATE_FORCED},
				AuthProviderId: authProviderID,
				Key:            "groups",
				Value:          "admin:org:all",
			},
			RoleName: "Admin",
		},
	}
	for _, adminGroup := range adminGroups {
		glog.Infof("Sending admin group request %+v\n", adminGroup)
		if err := m.centralClient.SendGroupRequest(context.Background(), adminGroup); err != nil {
			return errors.Wrapf(err, "creating admin group %+v for auth provider %s", adminGroup, authProviderID)
		}
		glog.Infof("Successfully created admin group %+v\n", adminGroup)
	}

	// Verify the groups are as expected.
	updatedGroups, err := getGroupsByAuthProviderID(authProviderID, m.centralClient)
	if err != nil {
		return err
	}

	glog.Infof("Groups for auth provider %s after updating:\n%+v\n", authProviderID, updatedGroups)

	return nil
}

func (m *migrator) storeAuthenticatedUsers() error {
	glog.Infof("Sending request to retrieve users from central %s", m.name)
	var usersResponse v1.GetUsersResponse
	if err := m.centralClient.SendRequestToCentral(context.Background(), nil, http.MethodGet, "/v1/users",
		&usersResponse); err != nil {
		return errors.Wrapf(err, "retrieving users for central %s", m.name)
	}
	glog.Infof("Received users from central %s:\n%+v\n", m.name, usersResponse)

	marshaller := jsonpb.Marshaler{Indent: "  "}
	path := path.Join(m.dir, fmt.Sprintf("%s-%s-users.json", m.name, m.id))
	f, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "creating file at path %s", path)
	}
	defer utils.IgnoreError(f.Close)
	glog.Infof("Created file to store central users at path %s", path)
	if err := marshaller.Marshal(f, &usersResponse); err != nil {
		return errors.Wrapf(err, "writing users for central %s to file %s:\n%+v", m.name, path, usersResponse)
	}
	glog.Infof("Wrote central users to file at path %s", path)
	return nil
}

func getGroupsByAuthProviderID(authProviderID string, client *client.Client) ([]*storage.Group, error) {
	var groups v1.GetGroupsResponse
	if err := client.SendRequestToCentral(context.Background(),
		nil,
		http.MethodGet, fmt.Sprintf("/v1/groups?authProviderId=%s", authProviderID), &groups); err != nil {
		return nil, errors.Wrapf(err, "retrieving groups associated with auth provider %q",
			authProviderID)
	}
	return groups.GetGroups(), nil
}
