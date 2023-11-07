package clusters

import (
	"net/http"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"

	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
)

func TestOCMProvider_Create(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterReq types.ClusterRequest
	}
	awsConfig := &config.AWSConfig{
		AccountID:       "",
		AccessKey:       "",
		SecretAccessKey: "",
	}
	osdCreateConfig := &config.DataplaneClusterConfig{
		OpenshiftVersion: "4.7",
	}
	cb := NewClusterBuilder(awsConfig, osdCreateConfig)

	internalID := "test-internal-id"
	externalID := "test-external-id"

	cr := types.ClusterRequest{
		CloudProvider:  "aws",
		Region:         "east-1",
		MultiAZ:        true,
		AdditionalSpec: nil,
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.ClusterSpec
		wantErr bool
	}{
		{
			name: "should return created cluster",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateClusterFunc: func(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error) {
						return clustersmgmtv1.NewCluster().ID(internalID).ExternalID(externalID).Build()
					},
				},
			},
			args: args{
				clusterReq: cr,
			},
			want: &types.ClusterSpec{
				InternalID:     internalID,
				ExternalID:     externalID,
				Status:         api.ClusterProvisioning,
				AdditionalInfo: nil,
			},
			wantErr: false,
		},
		{
			name: "should return error when create cluster failed from OCM",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateClusterFunc: func(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error) {
						return nil, errors.Errorf("failed to create cluster")
					},
				},
			},
			args:    args{clusterReq: cr},
			want:    nil,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, cb, &ocm.OCMConfig{})
			resp, err := p.Create(&test.args.clusterReq)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_CheckClusterStatus(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
	}

	internalID := "test-internal-id"
	externalID := "test-external-id"

	clusterFailedProvisioningErrorText := "cluster provisioning failed test message"

	spec := &types.ClusterSpec{
		InternalID:     internalID,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.ClusterSpec
		wantErr bool
	}{
		{
			name: "should return cluster status ready",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterFunc: func(clusterID string) (*clustersmgmtv1.Cluster, error) {
						sb := clustersmgmtv1.NewClusterStatus().State(clustersmgmtv1.ClusterStateReady)
						return clustersmgmtv1.NewCluster().Status(sb).ExternalID(externalID).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want: &types.ClusterSpec{
				InternalID:     internalID,
				ExternalID:     externalID,
				Status:         api.ClusterProvisioned,
				AdditionalInfo: nil,
			},
			wantErr: false,
		},
		{
			name: "should return cluster status failed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterFunc: func(clusterID string) (*clustersmgmtv1.Cluster, error) {
						sb := clustersmgmtv1.NewClusterStatus().State(clustersmgmtv1.ClusterStateError).ProvisionErrorMessage(clusterFailedProvisioningErrorText)
						return clustersmgmtv1.NewCluster().Status(sb).ExternalID(externalID).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want: &types.ClusterSpec{
				InternalID:     internalID,
				ExternalID:     externalID,
				Status:         api.ClusterFailed,
				StatusDetails:  clusterFailedProvisioningErrorText,
				AdditionalInfo: nil,
			},
			wantErr: false,
		},
		{
			name: "should return error when failed to get cluster from OCM",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterFunc: func(clusterID string) (*clustersmgmtv1.Cluster, error) {
						return nil, errors.Errorf("failed to get cluster")
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.CheckClusterStatus(test.args.clusterSpec)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_Delete(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
	}

	internalID := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalID,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should return true if cluster is not found from OCM",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteClusterFunc: func(clusterID string) (int, error) {
						return http.StatusNotFound, nil
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return false if the cluster still exists in OCM",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteClusterFunc: func(clusterID string) (int, error) {
						return http.StatusConflict, nil
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "should return error",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteClusterFunc: func(clusterID string) (int, error) {
						return 0, errors.Errorf("failed to delete cluster from OCM")
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.Delete(test.args.clusterSpec)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_GetClusterDNS(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
	}

	internalID := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalID,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	dns := "test.foo.bar.com"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "should return dns value from OCM",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterDNSFunc: func(clusterID string) (string, error) {
						return dns, nil
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want:    dns,
			wantErr: false,
		},
		{
			name: "should return error",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterDNSFunc: func(clusterID string) (string, error) {
						return "", errors.Errorf("failed to get dns value from OCM")
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.GetClusterDNS(test.args.clusterSpec)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_AddIdentityProvider(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec          *types.ClusterSpec
		identityProviderInfo types.IdentityProviderInfo
	}

	internalID := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalID,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	testIdpID := "test-idp-ocm-id"
	testIdpName := "test-idp-name"
	testIdpClientID := "test-client-id"
	testIdpClientSecret := "test-client-secret" // pragma: allowlist secret
	testIdpIssuer := "test-issuer"
	idpInfo := types.IdentityProviderInfo{OpenID: &types.OpenIDIdentityProviderInfo{
		Name:         testIdpName,
		ClientID:     testIdpClientID,
		ClientSecret: testIdpClientSecret, // pragma: allowlist secret
		Issuer:       testIdpIssuer,
	}}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.IdentityProviderInfo
		wantErr bool
	}{
		{
			name: "should create IDP",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateIdentityProviderFunc: func(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error) {
						return clustersmgmtv1.NewIdentityProvider().ID(testIdpID).Build()
					},
					GetIdentityProviderListFunc: func(clusterID string) (*clustersmgmtv1.IdentityProviderList, error) {
						return nil, errors.Errorf("this should not be called")
					},
				},
			},
			args: args{
				clusterSpec:          spec,
				identityProviderInfo: idpInfo,
			},
			want: &types.IdentityProviderInfo{
				OpenID: &types.OpenIDIdentityProviderInfo{
					ID:           testIdpID,
					Name:         testIdpName,
					ClientID:     testIdpClientID,
					ClientSecret: testIdpClientSecret, // pragma: allowlist secret
					Issuer:       testIdpIssuer,
				},
			},
			wantErr: false,
		},
		{
			name: "should not return error if IDP already exists",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateIdentityProviderFunc: func(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error) {
						return nil, errors.Errorf("idp already exists")
					},
					GetIdentityProviderListFunc: func(clusterID string) (*clustersmgmtv1.IdentityProviderList, error) {
						idp := clustersmgmtv1.NewIdentityProvider().ID(testIdpID).Name(testIdpName)
						return clustersmgmtv1.NewIdentityProviderList().Items(idp).Build()
					},
				},
			},
			args: args{
				clusterSpec:          spec,
				identityProviderInfo: idpInfo,
			},
			want: &types.IdentityProviderInfo{
				OpenID: &types.OpenIDIdentityProviderInfo{
					ID:           testIdpID,
					Name:         testIdpName,
					ClientID:     testIdpClientID,
					ClientSecret: testIdpClientSecret, // pragma: allowlist secret
					Issuer:       testIdpIssuer,
				},
			},
			wantErr: false,
		},
		{
			name: "should return error",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateIdentityProviderFunc: func(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error) {
						return nil, errors.Errorf("unexepcted error")
					},
					GetIdentityProviderListFunc: func(clusterID string) (*clustersmgmtv1.IdentityProviderList, error) {
						return nil, errors.Errorf("this should not be called")
					},
				},
			},
			args: args{
				clusterSpec:          spec,
				identityProviderInfo: idpInfo,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.AddIdentityProvider(test.args.clusterSpec, test.args.identityProviderInfo)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_ScaleUp(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
		increment   int
	}

	internalID := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalID,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.ClusterSpec
		wantErr bool
	}{
		{
			name: "should scaleUp",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ScaleUpComputeNodesFunc: func(clusterID string, increment int) (*clustersmgmtv1.Cluster, error) {
						return nil, nil
					},
				},
			},
			args: args{
				clusterSpec: spec,
				increment:   3,
			},
			want:    spec,
			wantErr: false,
		},
		{
			name: "should return error when failed to scale up",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ScaleUpComputeNodesFunc: func(clusterID string, increment int) (*clustersmgmtv1.Cluster, error) {
						return nil, errors.Errorf("failed to scale up")
					},
				},
			},
			args: args{
				clusterSpec: spec,
				increment:   3,
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.ScaleUp(test.args.clusterSpec, test.args.increment)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_ScaleDown(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
		decrement   int
	}

	internalID := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalID,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.ClusterSpec
		wantErr bool
	}{
		{
			name: "should scale down",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ScaleDownComputeNodesFunc: func(clusterID string, decrement int) (*clustersmgmtv1.Cluster, error) {
						return nil, nil
					},
				},
			},
			args: args{
				clusterSpec: spec,
				decrement:   3,
			},
			want:    spec,
			wantErr: false,
		},
		{
			name: "should return error when failed to scale up",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					ScaleDownComputeNodesFunc: func(clusterID string, decrement int) (*clustersmgmtv1.Cluster, error) {
						return nil, errors.Errorf("failed to scale up")
					},
				},
			},
			args: args{
				clusterSpec: spec,
				decrement:   3,
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.ScaleDown(test.args.clusterSpec, test.args.decrement)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_SetComputeNodes(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
		numNodes    int
	}

	internalID := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalID,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.ClusterSpec
		wantErr bool
	}{
		{
			name: "should scale down",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*clustersmgmtv1.Cluster, error) {
						return nil, nil
					},
				},
			},
			args: args{
				clusterSpec: spec,
				numNodes:    3,
			},
			want:    spec,
			wantErr: false,
		},
		{
			name: "should return error when failed to scale up",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					SetComputeNodesFunc: func(clusterID string, numNodes int) (*clustersmgmtv1.Cluster, error) {
						return nil, errors.Errorf("failed to scale up")
					},
				},
			},
			args: args{
				clusterSpec: spec,
				numNodes:    3,
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.SetComputeNodes(test.args.clusterSpec, test.args.numNodes)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_GetComputeNodes(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
	}

	internalID := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalID,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.ComputeNodesInfo
		wantErr bool
	}{
		{
			name: "should return compute nodes info",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterFunc: func(clusterID string) (*clustersmgmtv1.Cluster, error) {
						nodes := clustersmgmtv1.NewClusterNodes().Compute(6)
						return clustersmgmtv1.NewCluster().Nodes(nodes).Build()
					},
					GetExistingClusterMetricsFunc: func(clusterID string) (*v1.SubscriptionMetrics, error) {
						nodes := v1.NewClusterMetricsNodes().Compute(3)
						return v1.NewSubscriptionMetrics().Nodes(nodes).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want: &types.ComputeNodesInfo{
				Actual:  3,
				Desired: 6,
			},
			wantErr: false,
		},
		{
			name: "should return error when failed to get cluster",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterFunc: func(clusterID string) (*clustersmgmtv1.Cluster, error) {
						return nil, errors.Errorf("failed to get cluster info")
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "should return error when failed to get existing cluster metrics",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterFunc: func(clusterID string) (*clustersmgmtv1.Cluster, error) {
						nodes := clustersmgmtv1.NewClusterNodes().Compute(6)
						return clustersmgmtv1.NewCluster().Nodes(nodes).Build()
					},
					GetExistingClusterMetricsFunc: func(clusterID string) (*v1.SubscriptionMetrics, error) {
						return nil, errors.Errorf("failed to get existing cluster metrics")
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.GetComputeNodes(test.args.clusterSpec)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_InstallAddon(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
		addonID     string
	}

	internalID := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalID,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	testAddonID := "test-addon-id"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should create addon but not ready",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return clustersmgmtv1.NewAddOnInstallation().Build()
					},
					CreateAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						Expect(addonId).To(Equal(testAddonID))
						return clustersmgmtv1.NewAddOnInstallation().State(clustersmgmtv1.AddOnInstallationStateInstalling).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
				addonID:     testAddonID,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "should not create addon and it's ready",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						Expect(addonId).To(Equal(testAddonID))
						return clustersmgmtv1.NewAddOnInstallation().ID("test-addon-id").State(clustersmgmtv1.AddOnInstallationStateReady).Build()
					},
					CreateAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return nil, errors.Errorf("CreateAddon should not be called")
					},
				},
			},
			args: args{
				clusterSpec: spec,
				addonID:     testAddonID,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return error when failed to get addon",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return nil, errors.Errorf("failed to get addon")
					},
				},
			},
			args: args{
				clusterSpec: spec,
				addonID:     testAddonID,
			},
			wantErr: true,
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.installAddon(test.args.clusterSpec, test.args.addonID)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_InstallAddonWithParams(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
		addonID     string
		params      []types.Parameter
	}

	internalID := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalID,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	testAddonID := "test-addon-id"
	testParams := []types.Parameter{
		{
			ID:    "param1",
			Value: "param-value-1",
		},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should create addon with params if addon is not installed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return clustersmgmtv1.NewAddOnInstallation().Build()
					},
					CreateAddonWithParamsFunc: func(clusterId string, addonId string, params []types.Parameter) (*clustersmgmtv1.AddOnInstallation, error) {
						Expect(addonId).To(Equal(testAddonID))
						Expect(params).To(Equal(testParams))
						return clustersmgmtv1.NewAddOnInstallation().State(clustersmgmtv1.AddOnInstallationStateInstalling).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
				addonID:     testAddonID,
				params:      testParams,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "should not update addon if it's already installed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return clustersmgmtv1.NewAddOnInstallation().ID("test-addon-id").State(clustersmgmtv1.AddOnInstallationStateReady).Build()
					},
					UpdateAddonParametersFunc: func(clusterId string, addonId string, parameters []types.Parameter) (*clustersmgmtv1.AddOnInstallation, error) {
						Expect(addonId).To(Equal(testAddonID))
						Expect(parameters).To(Equal(testParams))
						return clustersmgmtv1.NewAddOnInstallation().State(clustersmgmtv1.AddOnInstallationStateReady).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
				addonID:     testAddonID,
				params:      testParams,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return error when failed to get addon",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return nil, errors.Errorf("failed to get addon")
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			wantErr: true,
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.installAddonWithParams(test.args.clusterSpec, test.args.addonID, test.args.params)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_GetCloudProviders(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}

	providerID1 := "provider-id-1"
	providerName1 := "provider-name-1"
	providerDisplayName1 := "provider-display-name-1"

	tests := []struct {
		name    string
		fields  fields
		want    *types.CloudProviderInfoList
		wantErr bool
	}{
		{
			name: "should return cloud providers when there are no cloud providers returned from ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetCloudProvidersFunc: func() (*clustersmgmtv1.CloudProviderList, error) {
						return clustersmgmtv1.NewCloudProviderList().Build()
					},
				},
			},
			want:    &types.CloudProviderInfoList{Items: nil},
			wantErr: false,
		},
		{
			name: "should return cloud providers when there are cloud providers returned from ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetCloudProvidersFunc: func() (*clustersmgmtv1.CloudProviderList, error) {
						p := clustersmgmtv1.NewCloudProvider().ID(providerID1).Name(providerName1).DisplayName(providerDisplayName1)
						return clustersmgmtv1.NewCloudProviderList().Items(p).Build()
					},
				},
			},
			want: &types.CloudProviderInfoList{Items: []types.CloudProviderInfo{{
				ID:          providerID1,
				Name:        providerName1,
				DisplayName: providerDisplayName1,
			}}},
			wantErr: false,
		},
		{
			name: "should return error when failed to get cloud providers",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetCloudProvidersFunc: func() (*clustersmgmtv1.CloudProviderList, error) {
						return nil, errors.Errorf("failed to get cloud providers")
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.GetCloudProviders()
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}

func TestOCMProvider_GetCloudProviderRegions(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}

	type args struct {
		providerInfo types.CloudProviderInfo
	}

	providerID1 := "provider-id-1"
	providerName1 := "provider-name-1"
	providerDisplayName1 := "provider-display-name-1"

	regionID1 := "region-id-1"
	regionName1 := "region-name-1"
	regionDisplayName1 := "region-display-name-1"
	regionSupportsMultiAZ1 := true

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.CloudProviderRegionInfoList
		wantErr bool
	}{
		{
			name: "should return cloud providers when there are no cloud providers returned from ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetRegionsFunc: func(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error) {
						Expect(provider.ID()).To(Equal(providerID1))
						Expect(provider.Name()).To(Equal(providerName1))
						Expect(provider.DisplayName()).To(Equal(providerDisplayName1))
						return clustersmgmtv1.NewCloudRegionList().Build()
					},
				},
			},
			args: args{providerInfo: types.CloudProviderInfo{
				ID:          providerID1,
				Name:        providerName1,
				DisplayName: providerDisplayName1,
			}},
			want:    &types.CloudProviderRegionInfoList{Items: nil},
			wantErr: false,
		},
		{
			name: "should return cloud providers when there are cloud providers returned from ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetRegionsFunc: func(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error) {
						Expect(provider.ID()).To(Equal(providerID1))
						Expect(provider.Name()).To(Equal(providerName1))
						Expect(provider.DisplayName()).To(Equal(providerDisplayName1))
						p := clustersmgmtv1.NewCloudProvider().ID(providerID1)
						r := clustersmgmtv1.NewCloudRegion().ID(regionID1).CloudProvider(p).Name(regionName1).DisplayName(regionDisplayName1).SupportsMultiAZ(regionSupportsMultiAZ1)
						return clustersmgmtv1.NewCloudRegionList().Items(r).Build()
					},
				},
			},
			args: args{providerInfo: types.CloudProviderInfo{
				ID:          providerID1,
				Name:        providerName1,
				DisplayName: providerDisplayName1,
			}},
			want: &types.CloudProviderRegionInfoList{
				Items: []types.CloudProviderRegionInfo{
					{
						ID:              regionID1,
						CloudProviderID: providerID1,
						Name:            regionName1,
						DisplayName:     regionDisplayName1,
						SupportsMultiAZ: regionSupportsMultiAZ1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should return error when failed to get cloud provider regions",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetRegionsFunc: func(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error) {
						return nil, errors.Errorf("failed get cloud provider regions")
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.GetCloudProviderRegions(test.args.providerInfo)
			Expect(resp).To(Equal(test.want))
			if test.wantErr {
				Expect(err).NotTo(BeNil())
			}
		})
	}
}
