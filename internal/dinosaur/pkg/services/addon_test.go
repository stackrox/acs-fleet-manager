package services

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
	addonsmgmtv1 "github.com/openshift-online/ocm-sdk-go/addonsmgmt/v1"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

func TestAddonProvisioner_Provision(t *testing.T) {
	RegisterTestingT(t)

	type fields struct {
		ocmClient *ocm.ClientMock
	}
	type args struct {
		cluster api.Cluster
		addons  []gitops.AddonConfig
	}
	tests := []struct {
		name    string
		setup   func()
		fields  fields
		args    args
		wantErr bool
		want    func(mock *ocm.ClientMock)
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
				},
			},
			args: args{
				addons: []gitops.AddonConfig{
					{
						ID: "acs-fleetshard",
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(len(mock.GetAddonInstallationCalls())).To(Equal(1))
			},
		},
		{
			name: "should return error when ocmClient.GetAddonInstallation returns error",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonInstallationFunc: func(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *errors.ServiceError) {
						return nil, errors.GeneralError("test")
					},
				},
			},
			args: args{
				addons: []gitops.AddonConfig{
					{
						ID: "acs-fleetshard",
					},
				},
			},
			wantErr: true,
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
				},
			},
			args: args{
				addons: []gitops.AddonConfig{
					{
						ID: "acs-fleetshard",
					},
				},
			},
			wantErr: true,
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
				},
			},
			args: args{
				addons: []gitops.AddonConfig{
					{
						ID: "alpha",
					},
					{
						ID: "beta",
					},
				},
			},
			wantErr: true,
			want: func(mock *ocm.ClientMock) {
				Expect(len(mock.CreateAddonInstallationCalls())).To(Equal(1))
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
				},
			},
			args: args{
				addons: []gitops.AddonConfig{
					{
						ID: "alpha",
					},
					{
						ID: "beta",
					},
				},
			},
			wantErr: true,
			want: func(mock *ocm.ClientMock) {
				Expect(len(mock.CreateAddonInstallationCalls())).To(Equal(2))
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
				},
			},
			args: args{
				addons: []gitops.AddonConfig{
					{
						ID:      "acs-fleetshard",
						Version: "0.2.0",
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(len(mock.UpdateAddonInstallationCalls())).To(BeZero())
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
					GetAddonVersionFunc: func(addonID string, version string) (*addonsmgmtv1.AddonVersion, error) {
						return addonsmgmtv1.NewAddonVersion().
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
				addons: []gitops.AddonConfig{
					{
						ID:      "acs-fleetshard",
						Version: "0.2.0",
						Parameters: map[string]string{
							"acscsEnvironment": "test",
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(len(mock.UpdateAddonInstallationCalls())).To(BeZero())
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
					GetAddonVersionFunc: func(addonID string, version string) (*addonsmgmtv1.AddonVersion, error) {
						return addonsmgmtv1.NewAddonVersion().
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
				addons: []gitops.AddonConfig{
					{
						ID:      "acs-fleetshard",
						Version: "0.2.0",
						Parameters: map[string]string{
							"acscsEnvironment": "test",
						},
					},
				},
			},
			wantErr: true,
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
					GetAddonVersionFunc: func(addonID string, version string) (*addonsmgmtv1.AddonVersion, error) {
						return addonsmgmtv1.NewAddonVersion().
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
				addons: []gitops.AddonConfig{
					{
						ID:      "acs-fleetshard",
						Version: "0.2.0",
						Parameters: map[string]string{
							"acscsEnvironment": "outdated",
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(len(mock.UpdateAddonInstallationCalls())).To(Equal(1))
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
					GetAddonVersionFunc: func(addonID string, version string) (*addonsmgmtv1.AddonVersion, error) {
						return addonsmgmtv1.NewAddonVersion().
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
				addons: []gitops.AddonConfig{
					{
						ID:      "acs-fleetshard",
						Version: "0.3.0",
						Parameters: map[string]string{
							"acscsEnvironment": "test",
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(len(mock.UpdateAddonInstallationCalls())).To(Equal(1))
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
					GetAddonVersionFunc: func(addonID string, version string) (*addonsmgmtv1.AddonVersion, error) {
						return addonsmgmtv1.NewAddonVersion().
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
				addons: []gitops.AddonConfig{
					{
						ID:      "acs-fleetshard",
						Version: "0.2.0",
						Parameters: map[string]string{
							"acscsEnvironment": "test",
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(len(mock.UpdateAddonInstallationCalls())).To(Equal(1))
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
					GetAddonVersionFunc: func(addonID string, version string) (*addonsmgmtv1.AddonVersion, error) {
						return addonsmgmtv1.NewAddonVersion().
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
				addons: []gitops.AddonConfig{
					{
						ID:      "acs-fleetshard",
						Version: "0.2.0",
						Parameters: map[string]string{
							"acscsEnvironment": "test",
						},
					},
				},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(len(mock.UpdateAddonInstallationCalls())).To(Equal(1))
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
					GetAddonVersionFunc: func(addonID string, version string) (*addonsmgmtv1.AddonVersion, error) {
						return addonsmgmtv1.NewAddonVersion().
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
				addons: []gitops.AddonConfig{},
			},
			want: func(mock *ocm.ClientMock) {
				Expect(len(mock.DeleteAddonInstallationCalls())).To(Equal(1))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			p := &AddonProvisioner{
				ocmClient: tt.fields.ocmClient,
			}
			err := p.Provision(tt.args.cluster, tt.args.addons)
			if tt.wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).To(Not(HaveOccurred()))
			}
			if tt.want != nil {
				tt.want(tt.fields.ocmClient)
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
			}, []gitops.AddonConfig{
				{
					ID:      "acs-fleetshard",
					Version: "0.2.0",
				},
			})
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(ocmMock.UpdateAddonInstallationCalls())).To(BeZero())
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
		}
		p := &AddonProvisioner{
			ocmClient: &mock,
		}
		err := p.Provision(api.Cluster{}, []gitops.AddonConfig{
			{
				ID:      "acs-fleetshard",
				Version: "0.2.0",
			},
		})
		Expect(err).To(Not(HaveOccurred()))
		Expect(len(mock.UpdateAddonInstallationCalls())).To(BeZero())
	})
}

func addonsJSON(addons []dbapi.AddonInstallation) api.JSON {
	result, err := json.Marshal(addons)
	Expect(err).To(Not(HaveOccurred()))
	return result
}
