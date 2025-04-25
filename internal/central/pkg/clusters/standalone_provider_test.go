package clusters

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	mocket "github.com/selvatico/go-mocket"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

func TestStandaloneProvider_GetCloudProviders(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	tests := []struct {
		name    string
		fields  fields
		want    *types.CloudProviderInfoList
		wantErr bool
		setupFn func()
	}{
		{
			name:    "receives an error when database query fails",
			wantErr: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithError(errors.New("some-error"))
			},
		},
		{
			name:    "returns an empty list when no standalone clusters exists",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			want: &types.CloudProviderInfoList{
				Items: []types.CloudProviderInfo{},
			},
		},
		{
			name:    "returns the list of cloud providers",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{{"cloud_provider": "aws"}, {"cloud_provider": "azure"}})
			},
			want: &types.CloudProviderInfoList{
				Items: []types.CloudProviderInfo{
					{
						ID:          "aws",
						Name:        "aws",
						DisplayName: "aws",
					},
					{
						ID:          "azure",
						Name:        "azure",
						DisplayName: "azure",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			test.setupFn()
			provider := newStandaloneProvider(test.fields.connectionFactory, config.NewDataplaneClusterConfig())
			resp, err := provider.GetCloudProviders()
			Expect(test.wantErr).To(Equal(err != nil))
			if !test.wantErr {
				Expect(resp.Items).To(Equal(test.want.Items))
			}
		})
	}
}

func TestStandaloneProvider_GetCloudProviderRegions(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	tests := []struct {
		name    string
		fields  fields
		want    *types.CloudProviderRegionInfoList
		wantErr bool
		setupFn func()
	}{
		{
			name:    "receives an error when database query fails",
			wantErr: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithError(errors.New("some-error"))
			},
		},
		{
			name:    "returns an empty list when no standalone clusters in a given cloud provider exists",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			want: &types.CloudProviderRegionInfoList{
				Items: []types.CloudProviderRegionInfo{},
			},
		},
		{
			name:    "returns the list of cloud providers regions",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{{"region": "af-east-1", "multi_az": false}, {"region": "eu-central-0", "multi_az": true}})
			},
			want: &types.CloudProviderRegionInfoList{
				Items: []types.CloudProviderRegionInfo{
					{
						ID:              "af-east-1",
						Name:            "af-east-1",
						DisplayName:     "af-east-1",
						CloudProviderID: "aws",
						SupportsMultiAZ: false,
					},
					{
						ID:              "eu-central-0",
						Name:            "eu-central-0",
						DisplayName:     "eu-central-0",
						CloudProviderID: "aws",
						SupportsMultiAZ: true,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			test.setupFn()
			provider := newStandaloneProvider(test.fields.connectionFactory, config.NewDataplaneClusterConfig())
			resp, err := provider.GetCloudProviderRegions(types.CloudProviderInfo{ID: "aws"})
			Expect(test.wantErr).To(Equal(err != nil))
			if !test.wantErr {
				Expect(resp.Items).To(Equal(test.want.Items))
			}
		})
	}
}

func TestStandaloneProvider_buildOpenIDPClientSecret(t *testing.T) {
	type args struct {
		idpProviderInfo types.IdentityProviderInfo
	}

	tests := []struct {
		name string
		args args
		want *v1.Secret
	}{
		{
			name: "buids a k8s secret with a given client secret",
			args: args{
				idpProviderInfo: types.IdentityProviderInfo{
					OpenID: &types.OpenIDIdentityProviderInfo{
						ClientSecret: "some-client-secret", // pragma: allowlist secret
					},
				},
			},
			want: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metav1.SchemeGroupVersion.Version,
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      centralSREOpenIDPSecretName,
					Namespace: "openshift-config",
				},
				Type: v1.SecretTypeOpaque,
				StringData: map[string]string{
					"clientSecret": "some-client-secret", // pragma: allowlist secret
				},
			},
		},
		{
			name: "buids a k8s secret with another given client secret",
			args: args{
				idpProviderInfo: types.IdentityProviderInfo{
					OpenID: &types.OpenIDIdentityProviderInfo{
						ClientSecret: "some-other-client-secret", // pragma: allowlist secret
					},
				},
			},
			want: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metav1.SchemeGroupVersion.Version,
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      centralSREOpenIDPSecretName,
					Namespace: "openshift-config",
				},
				Type: v1.SecretTypeOpaque,
				StringData: map[string]string{
					"clientSecret": "some-other-client-secret", // pragma: allowlist secret
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(db.NewMockConnectionFactory(nil), config.NewDataplaneClusterConfig())
			secret := provider.buildOpenIDPClientSecret(test.args.idpProviderInfo)
			Expect(secret).To(Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildIdentityProviderResource(t *testing.T) {
	type args struct {
		idpProviderInfo types.IdentityProviderInfo
	}

	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "buids a k8s secret with a given client secret",
			args: args{
				idpProviderInfo: types.IdentityProviderInfo{
					OpenID: &types.OpenIDIdentityProviderInfo{
						ClientSecret: "some-client-secret", // pragma: allowlist secret
						ID:           "some-id",
						Name:         "some-name",
						ClientID:     "some-client-id",
						Issuer:       "some-issuer",
					},
				},
			},
			want: map[string]interface{}{
				"apiVersion": "config.openshift.io/v1",
				"kind":       "OAuth",
				"metadata": map[string]string{
					"name": "cluster",
				},
				"spec": map[string]interface{}{
					"identityProviders": []map[string]interface{}{
						{
							"name":          "some-name",
							"mappingMethod": "claim",
							"type":          "OpenID",
							"openID": map[string]interface{}{
								"clientID": "some-client-id",
								"issuer":   "some-issuer",
								"clientSecret": map[string]string{
									"name": centralSREOpenIDPSecretName,
								},
								"claims": map[string][]string{
									"email":             {"email"},
									"preferredUsername": {"preferred_username"},
									"last_name":         {"preferred_username"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "buids a k8s secret with another given client secret",
			args: args{
				idpProviderInfo: types.IdentityProviderInfo{
					OpenID: &types.OpenIDIdentityProviderInfo{
						ClientSecret: "some-other-client-secret", // pragma: allowlist secret
						ID:           "some-id-1",
						Name:         "some-name-1",
						ClientID:     "some-client-id-1",
						Issuer:       "some-issuer-1",
					},
				},
			},
			want: map[string]interface{}{
				"apiVersion": "config.openshift.io/v1",
				"kind":       "OAuth",
				"metadata": map[string]string{
					"name": "cluster",
				},
				"spec": map[string]interface{}{
					"identityProviders": []map[string]interface{}{
						{
							"name":          "some-name-1",
							"mappingMethod": "claim",
							"type":          "OpenID",
							"openID": map[string]interface{}{
								"clientID": "some-client-id-1",
								"issuer":   "some-issuer-1",
								"clientSecret": map[string]string{
									"name": centralSREOpenIDPSecretName,
								},
								"claims": map[string][]string{
									"email":             {"email"},
									"preferredUsername": {"preferred_username"},
									"last_name":         {"preferred_username"},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(db.NewMockConnectionFactory(nil), config.NewDataplaneClusterConfig())
			secret := provider.buildIdentityProviderResource(test.args.idpProviderInfo)
			Expect(secret).To(Equal(test.want))
		})
	}
}
