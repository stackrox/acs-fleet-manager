package services

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	ocmImpl "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
	ocm "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/mocks"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

func TestAddonProvisioner_Provision(t *testing.T) {
	RegisterTestingT(t)
	type statusUpdate struct {
		clusterName string
		addonID     string
		status      metrics.AddonStatus
	}
	type fields struct {
		ocmClient *ocm.ClientMock
	}
	type args struct {
		cluster       api.Cluster
		clusterConfig gitops.DataPlaneClusterConfig
	}
	tests := []struct {
		name         string
		setup        func()
		fields       fields
		args         args
		wantErr      bool
		wantStatuses []statusUpdate
		want         func(mock *ocm.ClientMock)
	}{
		{
			name:    "should return no error when no addons have to be installed",
			wantErr: false,
		},
		{
			name: "should install addon when not installed in ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						return nil, errors.NotFound("")
					},
					CreateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
					GetAddonFunc: func(addonID string) (*clustersmgmtv1.AddOn, error) {
						return clustersmgmtv1.NewAddOn().
							ID(addonID).
							Version(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							Build()
					},
				},
			},
			args: args{
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID: "acs-fleetshard",
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.GetAddonInstallationCalls()).To(HaveLen(1))
			},
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonUpgrade},
			},
		},
		{
			name: "should return error when ocmClient.GetAddonInstallation returns error",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
					GetAddonFunc: func(addonID string) (*clustersmgmtv1.AddOn, error) {
						return clustersmgmtv1.NewAddOn().
							ID(addonID).
							Version(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							Build()
					},
				},
			},
			args: args{
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID: "acs-fleetshard",
						},
					},
				},
			},
			wantErr: true,
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonUnhealthy},
			},
		},
		{
			name: "should return error when ocmClient.CreateAddonInstallation returns error",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						return nil, errors.NotFound("")
					},
					CreateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return errors.GeneralError("test")
					},
					GetAddonFunc: func(addonID string) (*clustersmgmtv1.AddOn, error) {
						return clustersmgmtv1.NewAddOn().
							ID(addonID).
							Version(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							Build()
					},
				},
			},
			args: args{
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID: "acs-fleetshard",
						},
					},
				},
			},
			wantErr: true,
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonUnhealthy},
			},
		},
		{
			name: "should install one addon if failed to request another one from ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						if addonID == "beta" {
							return nil, errors.NotFound("")
						}
						return nil, errors.GeneralError("test")
					},
					CreateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
					GetAddonFunc: func(addonID string) (*clustersmgmtv1.AddOn, error) {
						return clustersmgmtv1.NewAddOn().
							ID(addonID).
							Version(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							Build()
					},
				},
			},
			args: args{
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID: "alpha",
						},
						{
							ID: "beta",
						},
					},
				},
			},
			wantErr: true,
			want: func(mock *ocm.ClientMock) {
				Expect(mock.CreateAddonInstallationCalls()).To(HaveLen(1))
			},
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "alpha", status: metrics.AddonUnhealthy},
				{clusterName: "acs-dev-dp-01", addonID: "beta", status: metrics.AddonUpgrade},
			},
		},
		{
			name: "should install one addon if can't create another one in ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						return nil, errors.NotFound("")
					},
					CreateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						if addon.Addon().ID() == "alpha" {
							return errors.GeneralError("test")
						}
						return nil
					},
					GetAddonFunc: func(addonID string) (*clustersmgmtv1.AddOn, error) {
						return clustersmgmtv1.NewAddOn().
							ID(addonID).
							Version(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							Build()
					},
				},
			},
			args: args{
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID: "alpha",
						},
						{
							ID: "beta",
						},
					},
				},
			},
			wantErr: true,
			want: func(mock *ocm.ClientMock) {
				Expect(mock.CreateAddonInstallationCalls()).To(HaveLen(2))
			},
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "alpha", status: metrics.AddonUnhealthy},
				{clusterName: "acs-dev-dp-01", addonID: "beta", status: metrics.AddonUpgrade},
			},
		},
		{
			name: "should NOT upgrade when no addons installed yet",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						return clustersmgmtv1.NewAddOnVersion().ID("0.2.0").Build()
					},
				},
			},
			args: args{
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.2.0",
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.UpdateAddonInstallationCalls()).To(BeEmpty())
			},
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonUpgrade},
			},
		},
		{
			name: "should NOT upgrade when the version in config didn't change",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							State(clustersmgmtv1.AddOnInstallationStateReady).
							Parameters(clustersmgmtv1.NewAddOnInstallationParameterList().Items(clustersmgmtv1.NewAddOnInstallationParameter().ID("acscsEnvironment").Value("test"))).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						return clustersmgmtv1.NewAddOnVersion().
							ID("0.2.0").
							SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
							PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
							Build()
					},
				},
			},
			args: args{
				cluster: api.Cluster{
					Addons: addonsJSON([]dbapi.AddonInstallation{
						{
							ID:                  "acs-fleetshard",
							Version:             "0.2.0",
							SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
							PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
							ParametersSHA256Sum: "2b44291c69be83a96af144cc390b7919aebf6421d3d9a976543198032f257a48", // pragma: allowlist secret
						},
					}),
				},
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.2.0",
							Parameters: map[string]string{
								"acscsEnvironment": "test",
							},
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.UpdateAddonInstallationCalls()).To(BeEmpty())
			},
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonHealthy},
			},
		},
		{
			name: "should return error when checksum mismatch",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							State(clustersmgmtv1.AddOnInstallationStateReady).
							Parameters(clustersmgmtv1.NewAddOnInstallationParameterList().Items(clustersmgmtv1.NewAddOnInstallationParameter().ID("acscsEnvironment").Value("test"))).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						return clustersmgmtv1.NewAddOnVersion().
							ID("0.2.0").
							SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
							PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
							Build()
					},
				},
			},
			args: args{
				cluster: api.Cluster{
					Addons: addonsJSON([]dbapi.AddonInstallation{
						{
							ID:                  "acs-fleetshard",
							Version:             "0.2.0",
							SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
							PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
							ParametersSHA256Sum: "f54d2c5cb370f4f87a31ccd8f72d97a85d89838720bd69278d1d40ee1cea00dc", // pragma: allowlist secret
						},
					}),
				},
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.2.0",
							Parameters: map[string]string{
								"acscsEnvironment": "test",
							},
						},
					},
				},
			},
			wantErr: true,
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonUnhealthy},
			},
		},
		{
			name: "should upgrade when parameters changed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							State(clustersmgmtv1.AddOnInstallationStateReady).
							Parameters(clustersmgmtv1.NewAddOnInstallationParameterList().Items(clustersmgmtv1.NewAddOnInstallationParameter().ID("acscsEnvironment").Value("test"))).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						return clustersmgmtv1.NewAddOnVersion().
							ID("0.2.0").
							SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
							PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
							Build()
					},
					UpdateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
				},
			},
			args: args{
				cluster: api.Cluster{
					Addons: addonsJSON([]dbapi.AddonInstallation{
						{
							ID:                  "acs-fleetshard",
							Version:             "0.2.0",
							SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
							PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
							ParametersSHA256Sum: "2b44291c69be83a96af144cc390b7919aebf6421d3d9a976543198032f257a48", // pragma: allowlist secret
						},
					}),
				},
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.2.0",
							Parameters: map[string]string{
								"acscsEnvironment": "outdated",
							},
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.UpdateAddonInstallationCalls()).To(HaveLen(1))
			},
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonUpgrade},
			},
		},
		{
			name: "should upgrade when version changed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							State(clustersmgmtv1.AddOnInstallationStateReady).
							Parameters(clustersmgmtv1.NewAddOnInstallationParameterList().Items(clustersmgmtv1.NewAddOnInstallationParameter().ID("acscsEnvironment").Value("test"))).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						return clustersmgmtv1.NewAddOnVersion().
							ID("0.2.0").
							SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
							PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
							Build()
					},
					UpdateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
				},
			},
			args: args{
				cluster: api.Cluster{
					Addons: addonsJSON([]dbapi.AddonInstallation{
						{
							ID:                  "acs-fleetshard",
							Version:             "0.2.0",
							SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
							PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
							ParametersSHA256Sum: "3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c", // pragma: allowlist secret
						},
					}),
				},
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.3.0",
							Parameters: map[string]string{
								"acscsEnvironment": "test",
							},
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.UpdateAddonInstallationCalls()).To(HaveLen(1))
			},
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonUpgrade},
			},
		},
		{
			name: "should upgrade when sourceImage changed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							State(clustersmgmtv1.AddOnInstallationStateReady).
							Parameters(clustersmgmtv1.NewAddOnInstallationParameterList().Items(clustersmgmtv1.NewAddOnInstallationParameter().ID("acscsEnvironment").Value("test"))).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						return clustersmgmtv1.NewAddOnVersion().
							ID("0.2.0").
							SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
							PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
							Build()
					},
					UpdateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
				},
			},
			args: args{
				cluster: api.Cluster{
					Addons: addonsJSON([]dbapi.AddonInstallation{
						{
							ID:                  "acs-fleetshard",
							Version:             "0.2.0",
							SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:81eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
							PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
							ParametersSHA256Sum: "3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c", // pragma: allowlist secret
						},
					}),
				},
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.3.0",
							Parameters: map[string]string{
								"acscsEnvironment": "test",
							},
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.UpdateAddonInstallationCalls()).To(HaveLen(1))
			},
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonUpgrade},
			},
		},
		{
			name: "should upgrade when packageImage changed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							State(clustersmgmtv1.AddOnInstallationStateReady).
							Parameters(clustersmgmtv1.NewAddOnInstallationParameterList().Items(clustersmgmtv1.NewAddOnInstallationParameter().ID("acscsEnvironment").Value("test"))).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						return clustersmgmtv1.NewAddOnVersion().
							ID("0.2.0").
							SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
							PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:4e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
							Build()
					},
					UpdateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
				},
			},
			args: args{
				cluster: api.Cluster{
					Addons: addonsJSON([]dbapi.AddonInstallation{
						{
							ID:                  "acs-fleetshard",
							Version:             "0.2.0",
							SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
							PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
							ParametersSHA256Sum: "3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c", // pragma: allowlist secret
						},
					}),
				},
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.3.0",
							Parameters: map[string]string{
								"acscsEnvironment": "test",
							},
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.UpdateAddonInstallationCalls()).To(HaveLen(1))
			},
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonUpgrade},
			},
		},
		{
			name: "should uninstall when no addon declared in gitops",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							State(clustersmgmtv1.AddOnInstallationStateReady).
							Parameters(clustersmgmtv1.NewAddOnInstallationParameterList().Items(clustersmgmtv1.NewAddOnInstallationParameter().ID("acscsEnvironment").Value("test"))).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						return clustersmgmtv1.NewAddOnVersion().
							ID("0.2.0").
							SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
							PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
							Build()
					},
					DeleteAddonInstallationFunc: func(clusterID string, addonID string) error {
						return nil
					},
				},
			},
			args: args{
				cluster: api.Cluster{
					Addons: addonsJSON([]dbapi.AddonInstallation{
						{
							ID:                  "acs-fleetshard",
							Version:             "0.2.0",
							SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:81eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
							PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
							ParametersSHA256Sum: "3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c", // pragma: allowlist secret
						},
					}),
				},
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons:      []gitops.AddonConfig{},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.DeleteAddonInstallationCalls()).To(HaveLen(1))
			},
			wantStatuses: []statusUpdate{
				{clusterName: "acs-dev-dp-01", addonID: "acs-fleetshard", status: metrics.AddonHealthy},
			},
		},
		{
			name: "should install addon with default parameter",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						return nil, errors.NotFound("")
					},
					CreateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
					GetAddonFunc: func(addonID string) (*clustersmgmtv1.AddOn, error) {
						return clustersmgmtv1.NewAddOn().
							ID(addonID).
							Version(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							Parameters(clustersmgmtv1.NewAddOnParameterList().Items(clustersmgmtv1.NewAddOnParameter().ID("defaultParam").DefaultValue("123"))).
							Build()

					},
				},
			},
			args: args{
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID: "acs-fleetshard",
							Parameters: map[string]string{
								"customParam": "abc",
							},
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.CreateAddonInstallationCalls()).To(HaveLen(1))
				Expect(mock.CreateAddonInstallationCalls()[0].Addon.Parameters().Len()).To(Equal(2))
			},
		},
		{
			name: "should parameter defined in gitops take precedence on install ",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						return nil, errors.NotFound("")
					},
					CreateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
					GetAddonFunc: func(addonID string) (*clustersmgmtv1.AddOn, error) {
						return clustersmgmtv1.NewAddOn().
							ID(addonID).
							Version(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							Parameters(clustersmgmtv1.NewAddOnParameterList().Items(clustersmgmtv1.NewAddOnParameter().ID("param").DefaultValue("default"))).
							Build()

					},
				},
			},
			args: args{
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID: "acs-fleetshard",
							Parameters: map[string]string{
								"param": "custom",
							},
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.CreateAddonInstallationCalls()).To(HaveLen(1))
				Expect(mock.CreateAddonInstallationCalls()[0].Addon.Parameters().Len()).To(Equal(1))
				Expect(mock.CreateAddonInstallationCalls()[0].Addon.Parameters().Get(0).ID()).To(Equal("param"))
				Expect(mock.CreateAddonInstallationCalls()[0].Addon.Parameters().Get(0).Value()).To(Equal("custom"))
			},
		},
		{
			name: "should upgrade with default parameter",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							State(clustersmgmtv1.AddOnInstallationStateReady).
							Parameters(clustersmgmtv1.NewAddOnInstallationParameterList().Items(clustersmgmtv1.NewAddOnInstallationParameter().ID("acscsEnvironment").Value("test"))).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						return clustersmgmtv1.NewAddOnVersion().
							ID("0.2.0").
							SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
							PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
							Parameters(clustersmgmtv1.NewAddOnParameterList().Items(
								clustersmgmtv1.NewAddOnParameter().ID("defaultParam").DefaultValue("abc"))).
							Build()
					},
					UpdateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
				},
			},
			args: args{
				cluster: api.Cluster{
					Addons: addonsJSON([]dbapi.AddonInstallation{
						{
							ID:                  "acs-fleetshard",
							Version:             "0.2.0",
							SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
							PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
							ParametersSHA256Sum: "2b44291c69be83a96af144cc390b7919aebf6421d3d9a976543198032f257a48", // pragma: allowlist secret
						},
					}),
				},
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.2.0",
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.UpdateAddonInstallationCalls()).To(HaveLen(1))
				Expect(mock.UpdateAddonInstallationCalls()[0].Addon.Parameters().Len()).To(Equal(1))
				Expect(mock.UpdateAddonInstallationCalls()[0].Addon.Parameters().Get(0).ID()).To(Equal("defaultParam"))
				Expect(mock.UpdateAddonInstallationCalls()[0].Addon.Parameters().Get(0).Value()).To(Equal("abc"))
			},
		},
		{
			name: "should parameter defined in gitops take precedence on update",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							State(clustersmgmtv1.AddOnInstallationStateReady).
							Parameters(clustersmgmtv1.NewAddOnInstallationParameterList().Items(clustersmgmtv1.NewAddOnInstallationParameter().ID("acscsEnvironment").Value("test"))).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						return clustersmgmtv1.NewAddOnVersion().
							ID("0.2.0").
							SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
							PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
							Parameters(clustersmgmtv1.NewAddOnParameterList().Items(
								clustersmgmtv1.NewAddOnParameter().ID("param").DefaultValue("default"))).
							Build()
					},
					UpdateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
				},
			},
			args: args{
				cluster: api.Cluster{
					Addons: addonsJSON([]dbapi.AddonInstallation{
						{
							ID:                  "acs-fleetshard",
							Version:             "0.2.0",
							SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
							PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
							ParametersSHA256Sum: "2b44291c69be83a96af144cc390b7919aebf6421d3d9a976543198032f257a48", // pragma: allowlist secret
						},
					}),
				},
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.2.0",
							Parameters: map[string]string{
								"param": "custom",
							},
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.UpdateAddonInstallationCalls()).To(HaveLen(1))
				Expect(mock.UpdateAddonInstallationCalls()[0].Addon.Parameters().Len()).To(Equal(1))
				Expect(mock.UpdateAddonInstallationCalls()[0].Addon.Parameters().Get(0).ID()).To(Equal("param"))
				Expect(mock.UpdateAddonInstallationCalls()[0].Addon.Parameters().Get(0).Value()).To(Equal("custom"))
			},
		},
		{
			name: "should default parameters be taken from the actual gitops version",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						object, err := clustersmgmtv1.NewAddOnInstallation().
							ID(addonID).
							Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
							AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
							State(clustersmgmtv1.AddOnInstallationStateReady).
							Parameters(clustersmgmtv1.NewAddOnInstallationParameterList().Items(clustersmgmtv1.NewAddOnInstallationParameter().ID("acscsEnvironment").Value("test"))).
							Build()
						Expect(err).To(Not(HaveOccurred()))
						return object, nil
					},
					GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
						if version == "0.2.0" {
							return clustersmgmtv1.NewAddOnVersion().
								ID("0.2.0").
								SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
								PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
								Parameters(clustersmgmtv1.NewAddOnParameterList().Items(
									clustersmgmtv1.NewAddOnParameter().ID("deprecatedParam").DefaultValue("value"))).
								Build()
						} else if version == "0.3.0" {
							return clustersmgmtv1.NewAddOnVersion().
								ID("0.3.0").
								SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:81eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
								PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:4e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
								Build()
						}
						return nil, errors.NotFound("version not found")
					},
					UpdateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
						return nil
					},
				},
			},
			args: args{
				cluster: api.Cluster{
					Addons: addonsJSON([]dbapi.AddonInstallation{
						{
							ID:                  "acs-fleetshard",
							Version:             "0.2.0",
							SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
							PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
							ParametersSHA256Sum: "3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c", // pragma: allowlist secret
						},
					}),
				},
				clusterConfig: gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.3.0",
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(mock.UpdateAddonInstallationCalls()).To(HaveLen(1))
				Expect(mock.UpdateAddonInstallationCalls()[0].Addon.Parameters().Len()).To(Equal(0))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			var updates []statusUpdate
			p := &AddonProvisioner{
				ocmClient: tt.fields.ocmClient,
				updateAddonStatusMetricFunc: func(addonID, clusterName string, status metrics.AddonStatus) {
					updates = append(updates, statusUpdate{addonID: addonID, clusterName: clusterName, status: status})
				},
			}
			err := p.Provision(tt.args.cluster, tt.args.clusterConfig)
			if tt.wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).To(Not(HaveOccurred()))
			}
			if tt.want != nil {
				tt.want(tt.fields.ocmClient)
			}

			if tt.fields.ocmClient != nil {
				if len(tt.fields.ocmClient.UpdateAddonInstallationCalls()) > 0 {
					Expect(p.lastUpgradeRequestTime).NotTo(Equal(time.Time{}))
					Expect(p.lastStatusPerInstall).NotTo(BeEmpty())
				}
			}
			if tt.wantStatuses != nil {
				Expect(updates).To(Equal(tt.wantStatuses))
			}
		})
	}
}

func TestAddonProvisioner_Provision_NonFinalState(t *testing.T) {
	RegisterTestingT(t)
	tests := []struct {
		name  string
		state clustersmgmtv1.AddOnInstallationState
	}{
		{
			name: "should skip the update if there's no installation state",
		},
		{
			name:  "should skip the update if the addon is in deleting state",
			state: clustersmgmtv1.AddOnInstallationStateDeleting,
		},
		{
			name:  "should skip the update if the addon is in installing state",
			state: clustersmgmtv1.AddOnInstallationStateInstalling,
		},
		{
			name:  "should skip the update if the addon is in pending state",
			state: clustersmgmtv1.AddOnInstallationStatePending,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ocmMock := &ocm.ClientMock{
				GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
					builder := clustersmgmtv1.NewAddOnInstallation().
						ID(addonID).
						Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
						AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0"))
					if tt.state != "" {
						builder = builder.State(tt.state)
					}
					object, err := builder.Build()
					Expect(err).To(Not(HaveOccurred()))
					return object, nil
				},
				GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
					return clustersmgmtv1.NewAddOnVersion().ID("0.2.0").Build()
				},
			}
			p := &AddonProvisioner{
				ocmClient: ocmMock,
			}
			err := p.Provision(api.Cluster{
				Addons: addonsJSON([]dbapi.AddonInstallation{
					{
						ID:                  "acs-fleetshard",
						Version:             "0.2.0",
						SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
						PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
						ParametersSHA256Sum: "f54d2c5cb370f4f87a31ccd8f72d97a85d89838720bd69278d1d40ee1cea00dc", // pragma: allowlist secret
					},
				}),
			},
				gitops.DataPlaneClusterConfig{
					ClusterID:   "123456789abcdef",
					ClusterName: "acs-dev-dp-01",
					Addons: []gitops.AddonConfig{
						{
							ID:      "acs-fleetshard",
							Version: "0.2.0",
						},
					},
				})
			Expect(err).To(Not(HaveOccurred()))
			Expect(ocmMock.UpdateAddonInstallationCalls()).To(BeEmpty())
		})
	}
}

func TestAddonProvisioner_Provision_AutoUpgradeDisabled(t *testing.T) {
	t.Setenv("RHACS_ADDON_AUTO_UPGRADE", "false")
	t.Run("should NOT upgrade when auto upgrade feature is disabled", func(t *testing.T) {
		mock := ocm.ClientMock{
			GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
				object, err := clustersmgmtv1.NewAddOnInstallation().
					ID(addonID).
					Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
					AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
					Build()
				Expect(err).To(Not(HaveOccurred()))
				return object, nil
			},
			GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
				return clustersmgmtv1.NewAddOnVersion().ID("0.2.0").Build()
			},
		}
		p := &AddonProvisioner{
			ocmClient: &mock,
		}
		err := p.Provision(api.Cluster{}, gitops.DataPlaneClusterConfig{
			Addons: []gitops.AddonConfig{
				{
					ID:      "acs-fleetshard",
					Version: "0.2.0",
				},
			},
		})
		Expect(err).To(Not(HaveOccurred()))
		Expect(mock.UpdateAddonInstallationCalls()).To(BeEmpty())
	})
}

func TestAddonProvisioner_Provision_InheritFleetshardImageTag_Install(t *testing.T) {
	RegisterTestingT(t)

	ocmMock := &ocm.ClientMock{
		GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
			return nil, errors.NotFound("")
		},
		CreateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
			return nil
		},
		GetAddonFunc: func(addonID string) (*clustersmgmtv1.AddOn, error) {
			return clustersmgmtv1.NewAddOn().
				ID(addonID).
				Version(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
				Build()
		},
	}
	addonConfig := ocmImpl.AddonConfig{
		FleetshardSyncImageTag:        "0307e03",
		InheritFleetshardSyncImageTag: true,
	}
	p := &AddonProvisioner{
		ocmClient:      ocmMock,
		customizations: initCustomizations(addonConfig),
	}
	err := p.Provision(api.Cluster{}, gitops.DataPlaneClusterConfig{
		Addons: []gitops.AddonConfig{
			{
				ID: "acs-fleetshard-dev",
				Parameters: map[string]string{
					"fleetshardSyncImageTag": "inherit",
				},
			},
		},
	})
	Expect(err).To(Not(HaveOccurred()))
	Expect(ocmMock.CreateAddonInstallationCalls()).To(HaveLen(1))
	Expect(ocmMock.CreateAddonInstallationCalls()[0].Addon.Parameters().Len()).To(Equal(1))
	Expect(ocmMock.CreateAddonInstallationCalls()[0].Addon.Parameters().Get(0).ID()).To(Equal("fleetshardSyncImageTag"))
	Expect(ocmMock.CreateAddonInstallationCalls()[0].Addon.Parameters().Get(0).Value()).To(Equal("0307e03"))
}

func TestAddonProvisioner_Provision_InheritFleetshardImageTag_Upgrade(t *testing.T) {
	RegisterTestingT(t)

	ocmMock := &ocm.ClientMock{
		GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
			object, err := clustersmgmtv1.NewAddOnInstallation().
				ID(addonID).
				Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
				AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
				State(clustersmgmtv1.AddOnInstallationStateReady).
				Build()
			Expect(err).To(Not(HaveOccurred()))
			return object, nil
		},
		GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
			return clustersmgmtv1.NewAddOnVersion().
				ID("0.2.0").
				SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
				PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
				Build()
		},
		UpdateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
			return nil
		},
	}
	addonConfig := ocmImpl.AddonConfig{
		FleetshardSyncImageTag:        "0307e03",
		InheritFleetshardSyncImageTag: true,
	}
	p := &AddonProvisioner{
		ocmClient:      ocmMock,
		customizations: initCustomizations(addonConfig),
	}
	err := p.Provision(api.Cluster{
		Addons: addonsJSON([]dbapi.AddonInstallation{
			{
				ID:                  "acs-fleetshard",
				Version:             "0.2.0",
				SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
				PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
				ParametersSHA256Sum: "3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c", // pragma: allowlist secret
			},
		}),
	},
		gitops.DataPlaneClusterConfig{
			Addons: []gitops.AddonConfig{
				{
					ID:      "acs-fleetshard",
					Version: "0.3.0",
					Parameters: map[string]string{
						"fleetshardSyncImageTag": "inherit",
					},
				},
			},
		})

	Expect(err).To(Not(HaveOccurred()))
	Expect(ocmMock.UpdateAddonInstallationCalls()).To(HaveLen(1))
	Expect(ocmMock.UpdateAddonInstallationCalls()[0].Addon.Parameters().Len()).To(Equal(1))
	Expect(ocmMock.UpdateAddonInstallationCalls()[0].Addon.Parameters().Get(0).ID()).To(Equal("fleetshardSyncImageTag"))
	Expect(ocmMock.UpdateAddonInstallationCalls()[0].Addon.Parameters().Get(0).Value()).To(Equal("0307e03"))
}

func addonsJSON(addons []dbapi.AddonInstallation) api.JSON {
	result, err := json.Marshal(addons)
	Expect(err).To(Not(HaveOccurred()))
	return result
}

func TestAddonProvisioner_NewAddonProvisioner(t *testing.T) {
	RegisterTestingT(t)

	addonConfigPtr := &ocmImpl.AddonConfig{
		URL:          "https://addon-service.test",
		ClientID:     "addon-client-id",
		ClientSecret: "addon-client-secret", // pragma: allowlist secret
		SelfToken:    "addon-token",
	}

	baseConfigPtr := &ocmImpl.OCMConfig{
		BaseURL:      "https://base.test",
		ClientID:     "base-client-id",
		ClientSecret: "base-client-secret", // pragma: allowlist secret
		SelfToken:    "base-token",
	}

	_, err := NewAddonProvisioner(addonConfigPtr, baseConfigPtr)

	Expect(err).To(Not(HaveOccurred()))
	Expect(baseConfigPtr.BaseURL).To(Equal("https://base.test"))
	Expect(baseConfigPtr.ClientID).To(Equal("base-client-id"))
	Expect(baseConfigPtr.ClientSecret).To(Equal("base-client-secret"))
	Expect(baseConfigPtr.SelfToken).To(Equal("base-token"))
}

func TestAddonProvisioner_Provision_UpgradeBackoff(t *testing.T) {
	RegisterTestingT(t)

	ocmMock := &ocm.ClientMock{
		GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
			object, err := clustersmgmtv1.NewAddOnInstallation().
				ID(addonID).
				Addon(clustersmgmtv1.NewAddOn().ID(addonID)).
				AddonVersion(clustersmgmtv1.NewAddOnVersion().ID("0.2.0")).
				State(clustersmgmtv1.AddOnInstallationStateReady).
				Build()
			Expect(err).To(Not(HaveOccurred()))
			return object, nil
		},
		GetAddonVersionFunc: func(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error) {
			return clustersmgmtv1.NewAddOnVersion().
				ID("0.2.0").
				SourceImage("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac").
				PackageImage("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c").
				Build()
		},
		UpdateAddonInstallationFunc: func(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
			return nil
		},
	}
	addonConfig := ocmImpl.AddonConfig{
		FleetshardSyncImageTag:        "0307e03",
		InheritFleetshardSyncImageTag: true,
	}
	p := &AddonProvisioner{
		ocmClient:      ocmMock,
		customizations: initCustomizations(addonConfig),
		lastStatusPerInstall: map[string]metrics.AddonStatus{
			"cluster-id:acs-fleetshard": metrics.AddonUpgrade,
		},
		lastUpgradeRequestTime: time.Now(),
	}
	err := p.Provision(api.Cluster{
		Addons: addonsJSON([]dbapi.AddonInstallation{
			{
				ID:                  "acs-fleetshard",
				Version:             "0.2.0",
				SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
				PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
				ParametersSHA256Sum: "3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c", // pragma: allowlist secret
			},
		}),
	},
		gitops.DataPlaneClusterConfig{
			ClusterID: "cluster-id",
			Addons: []gitops.AddonConfig{
				{
					ID:      "acs-fleetshard",
					Version: "0.3.0",
					Parameters: map[string]string{
						"fleetshardSyncImageTag": "inherit",
					},
				},
			},
		})

	Expect(err).To(Not(HaveOccurred()))
	Expect(ocmMock.UpdateAddonInstallationCalls()).To(BeEmpty())
}
