package clusters

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/config"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha2 "github.com/operator-framework/api/pkg/operators/v1alpha2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stackrox/acs-fleet-manager/pkg/db"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	mocket "github.com/selvatico/go-mocket"
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
						ClientSecret: "some-client-secret",
					},
				},
			},
			want: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metav1.SchemeGroupVersion.Version,
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      dinosaurSREOpenIDPSecretName,
					Namespace: "openshift-config",
				},
				Type: v1.SecretTypeOpaque,
				StringData: map[string]string{
					"clientSecret": "some-client-secret",
				},
			},
		},
		{
			name: "buids a k8s secret with another given client secret",
			args: args{
				idpProviderInfo: types.IdentityProviderInfo{
					OpenID: &types.OpenIDIdentityProviderInfo{
						ClientSecret: "some-other-client-secret",
					},
				},
			},
			want: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metav1.SchemeGroupVersion.Version,
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      dinosaurSREOpenIDPSecretName,
					Namespace: "openshift-config",
				},
				Type: v1.SecretTypeOpaque,
				StringData: map[string]string{
					"clientSecret": "some-other-client-secret",
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
						ClientSecret: "some-client-secret",
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
									"name": dinosaurSREOpenIDPSecretName,
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
						ClientSecret: "some-other-client-secret",
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
									"name": dinosaurSREOpenIDPSecretName,
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

func TestStandaloneProvider_buildDinosaurOperatorNamespace(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *v1.Namespace
	}{
		{
			name: "buids a namespace with a given name",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DinosaurOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "namespace-name",
					},
				},
			},
			want: &v1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace-name",
				},
			},
		},
		{
			name: "buids a namespace with another given name",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DinosaurOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "another-namespace-name",
					},
				},
			},
			want: &v1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "another-namespace-name",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			namespace := provider.buildDinosaurOperatorNamespace()
			Expect(namespace).To(Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildDinosaurOperatorCatalogSource(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha1.CatalogSource
	}{
		{
			name: "buids a catalog source with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DinosaurOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "namespace-name",
						CatalogSourceNamespace: "catalog-namespace",
						IndexImage:             "index-image-1",
					},
				},
			},
			want: &operatorsv1alpha1.CatalogSource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.CatalogSourceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      dinosaurOperatorCatalogSourceName,
					Namespace: "catalog-namespace",
				},
				Spec: operatorsv1alpha1.CatalogSourceSpec{
					SourceType: operatorsv1alpha1.SourceTypeGrpc,
					Image:      "index-image-1",
				},
			},
		},
		{
			name: "buids a namespace with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DinosaurOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "another-namespace-name",
						CatalogSourceNamespace: "another-catalog-namespace",
						IndexImage:             "index-image-2",
					},
				},
			},
			want: &operatorsv1alpha1.CatalogSource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.CatalogSourceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      dinosaurOperatorCatalogSourceName,
					Namespace: "another-catalog-namespace",
				},
				Spec: operatorsv1alpha1.CatalogSourceSpec{
					SourceType: operatorsv1alpha1.SourceTypeGrpc,
					Image:      "index-image-2",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			catalogSource := provider.buildDinosaurOperatorCatalogSource()
			Expect(catalogSource).To(Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildDinosaurOperatorOperatorGroup(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha2.OperatorGroup
	}{
		{
			name: "buids a operator group with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DinosaurOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "namespace-name",
						CatalogSourceNamespace: "catalog-namespace",
						IndexImage:             "index-image-1",
					},
				},
			},
			want: &operatorsv1alpha2.OperatorGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha2.OperatorGroupKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      dinosaurOperatorOperatorGroupName,
					Namespace: "namespace-name",
				},
				Spec: operatorsv1alpha2.OperatorGroupSpec{},
			},
		},
		{
			name: "buids an operator group with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DinosaurOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "another-namespace-name",
						CatalogSourceNamespace: "another-catalog-namespace",
						IndexImage:             "index-image-2",
					},
				},
			},
			want: &operatorsv1alpha2.OperatorGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha2.OperatorGroupKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      dinosaurOperatorOperatorGroupName,
					Namespace: "another-namespace-name",
				},
				Spec: operatorsv1alpha2.OperatorGroupSpec{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			operatorGroup := provider.buildDinosaurOperatorOperatorGroup()
			Expect(operatorGroup).To(Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildDinosaurOperatorSubscription(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha1.Subscription
	}{
		{
			name: "buids a operator subscription with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DinosaurOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "namespace-name",
						CatalogSourceNamespace: "catalog-namespace",
						IndexImage:             "index-image-1",
						SubscriptionChannel:    "alpha",
						Package:                "package-1",
					},
				},
			},
			want: &operatorsv1alpha1.Subscription{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.SubscriptionKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      dinosaurOperatorSubscriptionName,
					Namespace: "namespace-name",
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					CatalogSource:          dinosaurOperatorCatalogSourceName,
					Channel:                "alpha",
					CatalogSourceNamespace: "catalog-namespace",
					InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
					Package:                "package-1",
				},
			},
		},
		{
			name: "buids an operator subscription with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DinosaurOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "another-namespace-name",
						CatalogSourceNamespace: "another-catalog-namespace",
						IndexImage:             "index-image-2",
						SubscriptionChannel:    "beta",
						Package:                "package-2",
					},
				},
			},
			want: &operatorsv1alpha1.Subscription{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.SubscriptionKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      dinosaurOperatorSubscriptionName,
					Namespace: "another-namespace-name",
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					CatalogSource:          dinosaurOperatorCatalogSourceName,
					Channel:                "beta",
					CatalogSourceNamespace: "another-catalog-namespace",
					InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
					Package:                "package-2",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			subscription := provider.buildDinosaurOperatorSubscription()
			Expect(subscription).To(Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildFleetshardOperatorNamespace(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *v1.Namespace
	}{
		{
			name: "buids a namespace with a given name",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					FleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "namespace-name",
					},
				},
			},
			want: &v1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace-name",
				},
			},
		},
		{
			name: "buids a namespace with another given name",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					FleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "another-namespace-name",
					},
				},
			},
			want: &v1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "another-namespace-name",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			namespace := provider.buildFleetShardOperatorNamespace()
			Expect(namespace).To(Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildFleetshardSyncSecret(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	type args struct {
		params []types.Parameter
	}

	tests := []struct {
		name   string
		args   args
		fields fields
		want   *v1.Secret
	}{
		{
			name: "buids a secrets with a given parameters",
			args: args{
				params: []types.Parameter{
					{
						Id:    "id-1",
						Value: "value-1",
					},
				},
			},
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					FleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "namespace-name",
					},
				},
			},
			want: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fleetShardOperatorParametersSecretName,
					Namespace: "namespace-name",
				},
				StringData: map[string]string{
					"id-1": "value-1",
				},
			},
		},
		{
			name: "buids a secret with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					FleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "another-namespace-name",
					},
				},
			},
			args: args{
				params: []types.Parameter{
					{
						Id:    "id",
						Value: "value",
					},
					{
						Id:    "id-2",
						Value: "value-2",
					},
				},
			},
			want: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fleetShardOperatorParametersSecretName,
					Namespace: "another-namespace-name",
				},
				StringData: map[string]string{
					"id":   "value",
					"id-2": "value-2",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			secret := provider.buildFleetShardSyncSecret(test.args.params)
			Expect(secret).To(Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildFleetshardCatalogSource(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha1.CatalogSource
	}{
		{
			name: "buids a catalog source with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					FleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "namespace-name",
						CatalogSourceNamespace: "catalog-namespace",
						IndexImage:             "index-image-1",
					},
				},
			},
			want: &operatorsv1alpha1.CatalogSource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.CatalogSourceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fleetShardOperatorCatalogSourceName,
					Namespace: "catalog-namespace",
				},
				Spec: operatorsv1alpha1.CatalogSourceSpec{
					SourceType: operatorsv1alpha1.SourceTypeGrpc,
					Image:      "index-image-1",
				},
			},
		},
		{
			name: "buids a namespace with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					FleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "another-namespace-name",
						CatalogSourceNamespace: "another-catalog-namespace",
						IndexImage:             "index-image-2",
					},
				},
			},
			want: &operatorsv1alpha1.CatalogSource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.CatalogSourceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fleetShardOperatorCatalogSourceName,
					Namespace: "another-catalog-namespace",
				},
				Spec: operatorsv1alpha1.CatalogSourceSpec{
					SourceType: operatorsv1alpha1.SourceTypeGrpc,
					Image:      "index-image-2",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			catalogSource := provider.buildFleetShardOperatorCatalogSource()
			Expect(catalogSource).To(Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildFleetshardOperatorGroup(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha2.OperatorGroup
	}{
		{
			name: "buids a operator group with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					FleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "namespace-name",
						CatalogSourceNamespace: "catalog-namespace",
						IndexImage:             "index-image-1",
					},
				},
			},
			want: &operatorsv1alpha2.OperatorGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha2.OperatorGroupKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fleetShardOperatorOperatorGroupName,
					Namespace: "namespace-name",
				},
				Spec: operatorsv1alpha2.OperatorGroupSpec{},
			},
		},
		{
			name: "buids an operator group with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					FleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "another-namespace-name",
						CatalogSourceNamespace: "another-catalog-namespace",
						IndexImage:             "index-image-2",
					},
				},
			},
			want: &operatorsv1alpha2.OperatorGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha2.OperatorGroupKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fleetShardOperatorOperatorGroupName,
					Namespace: "another-namespace-name",
				},
				Spec: operatorsv1alpha2.OperatorGroupSpec{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			operatorGroup := provider.buildFleetShardOperatorOperatorGroup()
			Expect(operatorGroup).To(Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildFleetshardOperatorSubscription(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha1.Subscription
	}{
		{
			name: "buids a operator subscription with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					FleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "namespace-name",
						CatalogSourceNamespace: "catalog-namespace",
						IndexImage:             "index-image-1",
						SubscriptionChannel:    "alpha",
						Package:                "package-1",
					},
				},
			},
			want: &operatorsv1alpha1.Subscription{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.SubscriptionKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fleetShardOperatorSubscriptionName,
					Namespace: "namespace-name",
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					CatalogSource:          fleetShardOperatorCatalogSourceName,
					Channel:                "alpha",
					CatalogSourceNamespace: "catalog-namespace",
					InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
					Package:                "package-1",
				},
			},
		},
		{
			name: "buids an operator subscription with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					FleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:              "another-namespace-name",
						CatalogSourceNamespace: "another-catalog-namespace",
						IndexImage:             "index-image-2",
						SubscriptionChannel:    "beta",
						Package:                "package-2",
					},
				},
			},
			want: &operatorsv1alpha1.Subscription{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.SubscriptionKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fleetShardOperatorSubscriptionName,
					Namespace: "another-namespace-name",
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					CatalogSource:          fleetShardOperatorCatalogSourceName,
					Channel:                "beta",
					CatalogSourceNamespace: "another-catalog-namespace",
					InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
					Package:                "package-2",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RegisterTestingT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			subscription := provider.buildFleetShardOperatorSubscription()
			Expect(subscription).To(Equal(test.want))
		})
	}
}
