package services

import (
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/clusters"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/clusters/types"

	. "github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
	mocket "github.com/selvatico/go-mocket"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

// This test file should act as a golden test file to show the usage of mocks

func Test_CachedCloudProviderWithRegions(t *testing.T) {
	type fields struct {
		providerFactory   clusters.ProviderFactory
		connectionFactory *db.ConnectionFactory
		cache             *cache.Cache
	}

	tests := []struct {
		name    string
		fields  fields
		setupFn func()
		wantErr bool
		want    []CloudProviderWithRegions
	}{
		{
			name: "successful get cloud provider regions from various provider types",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providerFactory: &clusters.ProviderFactoryMock{ // define a mock object here and provide stub implementation of the different services being called
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						if providerType == "ocm" {
							return &clusters.ProviderMock{
								GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
									return &types.CloudProviderInfoList{
										Items: []types.CloudProviderInfo{
											{
												ID:          "aws",
												Name:        "aws",
												DisplayName: "aws",
											},
										},
									}, nil
								},
								GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
									return &types.CloudProviderRegionInfoList{
										Items: []types.CloudProviderRegionInfo{
											{
												ID:              "af-east-1",
												CloudProviderID: providerInf.ID,
												Name:            "af-east-1",
												DisplayName:     "af-east-1",
											},
										},
									}, nil
								},
							}, nil
						}

						return &clusters.ProviderMock{
							GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
								return &types.CloudProviderInfoList{
									Items: []types.CloudProviderInfo{
										{
											ID:          "azure",
											Name:        "azure",
											DisplayName: "azure",
										},
										{
											ID:          "aws",
											Name:        "aws",
											DisplayName: "aws",
										},
									},
								}, nil
							},
							GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
								return &types.CloudProviderRegionInfoList{
									Items: []types.CloudProviderRegionInfo{
										{
											ID:              "af-east-1",
											CloudProviderID: providerInf.ID,
											Name:            "af-east-1",
											DisplayName:     "af-east-1",
										},
										{
											ID:              "af-south-1",
											CloudProviderID: providerInf.ID,
											Name:            "af-south-1",
											DisplayName:     "af-south-1",
										},
									},
								}, nil
							},
						}, nil
					}},
				cache: cache.New(5*time.Minute, 10*time.Minute),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().
					WithQuery("SELECT DISTINCT").
					WithReply([]map[string]interface{}{{"provider_type": "ocm"}, {"provider_type": "standalone"}})
			},
			wantErr: false,
			want: []CloudProviderWithRegions{
				{
					ID: "aws",
					RegionList: &types.CloudProviderRegionInfoList{
						Items: []types.CloudProviderRegionInfo{
							{
								CloudProviderID: "aws",
								Name:            "af-east-1",
								DisplayName:     "af-east-1",
								ID:              "af-east-1",
							},
							{
								CloudProviderID: "aws",
								Name:            "af-south-1",
								DisplayName:     "af-south-1",
								ID:              "af-south-1",
							},
						},
					},
				},
				{
					ID: "azure",
					RegionList: &types.CloudProviderRegionInfoList{
						Items: []types.CloudProviderRegionInfo{
							{
								CloudProviderID: "azure",
								Name:            "af-east-1",
								DisplayName:     "af-east-1",
								ID:              "af-east-1",
							},
							{
								CloudProviderID: "azure",
								Name:            "af-south-1",
								DisplayName:     "af-south-1",
								ID:              "af-south-1",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			tt.setupFn()
			p := cloudProvidersService{
				providerFactory:   tt.fields.providerFactory,
				cache:             tt.fields.cache,
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := p.GetCachedCloudProvidersWithRegions()
			Expect(tt.wantErr).To(Equal(err != nil))
			Expect(got).To(Equal(tt.want))
		})
	}
}
