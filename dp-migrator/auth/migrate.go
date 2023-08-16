// Package auth handles auth migrations for data plane clusters.
package auth

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/client"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/reconciler"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/utils"
)

// MigrateCommand provides the command to run the migration of authn/z resources on the dataplane clusters to support
// declarative configuration.
func MigrateCommand() *cobra.Command {
	m := migrator{}

	cmd := &cobra.Command{
		Use:   "authn-z",
		Short: "Run the migration of authn/z resources to declarative configuration",
		Long: `Run the migration of authn/z resources to declarative configuration.
For existing instances, the default authn/z resources (auth provider and groups) have been created imperatively via
API calls. This makes migrating things hard (that's why we have to write migrations like these).

While the secret has been created already containing the declarative configuration, they are currently not applied
since the auth provider cannot be created due to name clashes with the existing one.

The goal of this migration is to do the following:
* Retrieve potential custom groups added by users within the default auth provider.
* Remove the default auth provider that's created imperatively.
* Re-apply any custom groups that have been created previously.

The time until the auth provider is added after removal should be no more than 20 seconds, as this is the interval
Central will read and reconcile declarative configurations. The expectation is that the downtime should be less than 1
minute. It's worthwhile to note that users will have to re-login, as the auth provider ID changes, thus tokens issued
by the specific auth provider ID will be seen as invalid.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := m.Construct(); err != nil {
				if errors.Is(err, shared.ErrPausedReconciliation) {
					glog.Warningf("Central %s has reconciliation paused. It currently "+
						"CAN NOT be migrated and will be SKIPPED!", m.centralName)
					return nil
				}
				return errors.Wrap(err, "constructing command's dependencies")
			}
			if err := m.DoMigration(); err != nil {
				return errors.Wrapf(err, "migrating instance %q", m.centralName)
			}
			return nil
		},
		SilenceUsage: true,
	}

	cmd.Flags().StringVarP(&m.centralURL, "url", "u", "",
		`URL of the central instance which should be migrated. The format should be https://<hostname>:<port> and
should be the UI URL, not the data URL.`)
	cmd.Flags().StringVar(&m.centralName, "name", "", "name of the central instance")
	cmd.Flags().StringVar(&m.centralID, "id", "", "id of the central instance")
	cmd.Flags().StringVar(&m.centralIssuer, "issuer", "", "issuer for the auth provider of the central instance")
	cmd.Flags().BoolVar(&m.skipConfirmation, "skip-confirmation", false, "skip user confirmation before executing each step")
	cmd.Flags().BoolVar(&m.readOnly, "read-only", false, "read only auth provider and groups without making changes")

	utils.Must(cmd.MarkFlagRequired("url"))
	utils.Must(cmd.MarkFlagRequired("id"))
	utils.Must(cmd.MarkFlagRequired("name"))

	return cmd
}

type migrator struct {
	centralID        string
	centralName      string
	centralURL       string
	centralIssuer    string
	skipConfirmation bool
	readOnly         bool

	adminPassword    string
	parsedCentralURL *url.URL
	apiClient        *client.Client
}

func (m *migrator) Construct() error {
	parsedCentralURL, err := url.Parse(m.centralURL)
	if err != nil {
		return errors.Wrapf(err, "parsing central URL %q", m.centralURL)
	}
	m.parsedCentralURL = parsedCentralURL

	// Always enable the admin password for the specific central instance.
	adminPassword, err := shared.EnableAdminPassword(context.Background(), m.centralID, m.centralName, m.centralURL)
	if err != nil {
		return errors.Wrapf(err, "enabling admin password and basic auth provider for central %q", m.centralName)
	}

	m.adminPassword = adminPassword // pragma: allowlist secret

	// This is currently required due to the Central client requiring this to be injected. It will only be used for
	// logging purposes in case of errors. Ideally this shouldn't really be a dependency when creating the client.
	remoteCentral := private.ManagedCentral{Metadata: private.ManagedCentralAllOfMetadata{
		Name:      m.centralURL,
		Namespace: "default",
	}}
	m.apiClient = client.NewCentralClient(remoteCentral, m.centralURL, m.adminPassword)

	return nil
}

func (m *migrator) DoMigration() error {
	defer func() {
		utils.Must(shared.DisableAdminPassword(context.Background(), m.centralID, m.centralName))
	}()

	// 1. Retrieve all existing information from Central.
	// This includes:
	//	- the existing groups associated with the auth provider. Note that we will skip groups which are immutable,
	//	  since they are the "default" groups and are already covered by the declarative configuration created.
	defaultAuthProviderID, customGroups, err := m.retrieveAuthProviderAndGroups()
	if err != nil {
		return errors.Wrapf(err, "retrieving auth provider and groups for central %q", m.centralName)
	}

	// Short-circuit here: if the auth provider ID is the declarative auth provider ID, then we can assume
	// we have already migrated this instance.
	if defaultAuthProviderID == declarativeconfig.NewDeclarativeAuthProviderUUID(m.defaultAuthProviderName()).String() {
		glog.Infof("Central %s already uses declarative configuration for the authN/Z configuration, skipping..",
			m.centralName)
		return nil
	}

	if m.readOnly {
		glog.Infof("Read only mode selected, returning after retrieving all groups and auth provider")
		return nil
	}

	// 2. Delete the existing auth provider that has been created imperatively.
	waitForConfirmation("deleting the existing auth provider", m.skipConfirmation)
	if err := m.removeDefaultAuthProvider(defaultAuthProviderID); err != nil {
		return errors.Wrapf(err, "removing auth provider %q for central %q", defaultAuthProviderID, m.centralName)
	}

	// 3. Verify that the new auth provider will be created eventually.
	//    Note that we can query this via API since the UUID of the auth provider will be deterministic.
	waitForConfirmation("verifying the declarative auth provider exists", m.skipConfirmation)
	if err := m.verifyAuthProviderExists(); err != nil {
		return errors.Wrapf(err, "verifying auth provider exists for central %q", m.centralName)
	}

	// 4. Re-create the previously existing groups with references to the newly declarative created auth provider.
	waitForConfirmation("migrating custom groups to the new auth provider", m.skipConfirmation)
	if err := m.migrateCustomGroups(customGroups); err != nil {
		return errors.Wrapf(err, "migrating groups to new auth provider for central %q", m.centralName)
	}
	// That's it, we are done.
	return nil
}

func (m *migrator) retrieveAuthProviderAndGroups() (string, []*storage.Group, error) {
	glog.Infof("Retrieving login auth providers for central %q", m.centralName)
	authProviders, err := m.apiClient.GetLoginAuthProviders(context.Background())
	if err != nil {
		return "", nil, errors.Wrapf(err, "retrieving login auth providers for central %q", m.centralName)
	}

	glog.Infof("Received login auth providers from Central %q:\n%s", m.centralName, prettyPrintProto(authProviders))

	authProviderID, exists := m.getDefaultAuthProviderID(authProviders)
	if !exists {
		return "", nil, errors.Errorf("could not find default sso.r.c. auth provider for central %q", m.centralName)
	}

	groups, err := m.getGroupsForAuthProvider(authProviderID)
	if err != nil {
		return "", nil, errors.Wrapf(err, "retrieving groups for auth provider %q for central %q",
			authProviderID, m.centralName)
	}

	glog.Info("Filtering out groups that are non-default")
	var filteredGroups []*storage.Group
	for _, group := range groups {
		// ALLOW_MUTATE_FORCE will only be set for groups created imperatively.
		if group.GetProps().GetTraits().GetMutabilityMode() == storage.Traits_ALLOW_MUTATE_FORCED {
			continue
		}
		// We will also skip the default group since it will already be created by the declarative config.
		if group.GetProps().GetValue() == "" && group.GetProps().GetKey() == "" {
			continue
		}
		filteredGroups = append(filteredGroups, group)
	}
	glog.Infof("Groups after filtering out non-default ones: \n%s", prettyPrintGroups(filteredGroups))

	return authProviderID, filteredGroups, nil
}

func (m *migrator) removeDefaultAuthProvider(id string) error {
	glog.Infof("Removing auth provider %q for central %q", id, m.centralName)
	if err := m.apiClient.SendRequestToCentral(context.Background(), nil, http.MethodDelete,
		fmt.Sprintf("/v1/authProviders/%s?force=true", id), nil); err != nil {
		return errors.Wrapf(err, "attempting to delete auth provider %q for central %q", id, m.centralName)
	}

	glog.Infof("Successfully deleted auth provider %q for central %q", id, m.centralName)
	return nil
}

func (m *migrator) verifyAuthProviderExists() error {
	verified := concurrency.PollWithTimeout(func() bool {
		authProviders, err := m.apiClient.GetLoginAuthProviders(context.Background())
		if err != nil {
			glog.Errorf("Received an error when attempting to retrieve login auth providers for central %q",
				m.centralName)
			return false
		}
		glog.Infof("Received login auth providers for central %q:\n%s", m.centralName, prettyPrintProto(authProviders))
		authProviderID, exists := m.getDefaultAuthProviderID(authProviders)

		glog.Infof("Received auth provider exists (%t) and ID (%s)", exists, authProviderID)

		if exists && authProviderID == declarativeconfig.
			NewDeclarativeAuthProviderUUID(m.defaultAuthProviderName()).String() {
			glog.Infof("Found the auth provider ID we expect from the declarative provider")
			return true
		}
		return false
	}, 5*time.Second, 5*time.Minute)

	if verified {
		glog.Infof("Successfully verified that the declarative auth provider exists for central %q", m.centralName)
		return nil
	}

	glog.Infof("Failed to verify the declarative auth provider exists for central %q", m.centralName)
	return errors.Errorf("failed to verify that the declarative auth provider exists")
}

func (m *migrator) migrateCustomGroups(groups []*storage.Group) error {
	glog.Infof("Custom groups before adjusting the auth provider ID for central %q:\n%s", m.centralName,
		prettyPrintGroups(groups))
	declarativeAuthProviderUUID := declarativeconfig.NewDeclarativeAuthProviderUUID(m.defaultAuthProviderName()).String()
	glog.Infof("New auth provider ID: %q", declarativeAuthProviderUUID)

	for _, group := range groups {
		group.Props.AuthProviderId = declarativeAuthProviderUUID
		group.Props.Id = ""
		if err := m.apiClient.SendRequestToCentral(context.Background(), group, http.MethodPost, "/v1/groups",
			nil); err != nil {
			return errors.Wrapf(err, "creating group %s for auth provider %q for central %q",
				prettyPrintProto(group), declarativeAuthProviderUUID, m.centralName)
		}
	}

	glog.Infof("Verifying that groups are as expected")
	groups, err := m.getGroupsForAuthProvider(declarativeAuthProviderUUID)
	if err != nil {
		return errors.Wrapf(err, "retrieving groups for auth provider %q for central %q",
			declarativeAuthProviderUUID, m.centralName)
	}
	glog.Infof("Groups after migration:\n%s", prettyPrintGroups(groups))
	return nil
}

func (m *migrator) defaultAuthProviderName() string {
	return reconciler.AuthProviderName(
		private.ManagedCentral{
			Spec: private.ManagedCentralAllOfSpec{
				Auth: private.ManagedCentralAllOfSpecAuth{
					Issuer: m.centralIssuer,
				},
			},
		},
	)
}

func (m *migrator) getGroupsForAuthProvider(id string) ([]*storage.Group, error) {
	glog.Infof("Retrieving groups for auth provider %q", id)
	var groups v1.GetGroupsResponse
	if err := m.apiClient.SendRequestToCentral(context.Background(),
		nil,
		http.MethodGet, fmt.Sprintf("/v1/groups?authProviderId=%s", id), &groups); err != nil {
		return nil, errors.Wrapf(err, "retrieving groups associated with auth provider %q for central %q",
			id, m.centralName)
	}

	glog.Infof("Received groups associated with auth provider %q for Central %q:\n%s",
		id, m.centralName, prettyPrintProto(&groups))
	return groups.GetGroups(), nil
}

func (m *migrator) getDefaultAuthProviderID(authProviders *v1.GetLoginAuthProvidersResponse) (string, bool) {
	for _, provider := range authProviders.GetAuthProviders() {
		if provider.GetType() == "oidc" && provider.GetName() == m.defaultAuthProviderName() {
			glog.Infof("Found default sso.r.c auth provider for Central %q:\n%s", m.centralName, prettyPrintProto(provider))
			return provider.GetId(), true
		}
	}
	return "", false
}

func waitForConfirmation(step string, skip bool) {
	if skip {
		return
	}
	fmt.Printf("Press Enter to continue with %q\n", step)
	_, _ = fmt.Scanln()
}

func prettyPrintProto(msgs ...proto.Message) string {
	buf := &bytes.Buffer{}
	marshaller := jsonpb.Marshaler{
		Indent: "  ",
	}
	for _, msg := range msgs {
		utils.Must(marshaller.Marshal(buf, msg))
	}
	return buf.String()
}

func prettyPrintGroups(groups []*storage.Group) string {
	var res string
	for _, g := range groups {
		res += prettyPrintProto(g)
	}
	return res
}
