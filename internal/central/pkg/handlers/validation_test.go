package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/centrals/types"

	"github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
)

func Test_Validation_validateCentralClusterNameIsUnique(t *testing.T) {
	type args struct {
		centralService services.CentralService
		name           string
		context        context.Context
	}

	tests := []struct {
		name string
		arg  args
		want *errors.ServiceError
	}{
		{
			name: "throw an error when the CentralService call throws an error",
			arg: args{
				centralService: &services.CentralServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 4}, errors.GeneralError("count failed from database")
					},
				},
				name:    "some-name",
				context: context.TODO(),
			},
			want: errors.GeneralError("count failed from database"),
		},
		{
			name: "throw an error when name is already used",
			arg: args{
				centralService: &services.CentralServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 1}, nil
					},
				},
				name:    "duplicate-name",
				context: context.TODO(),
			},
			want: &errors.ServiceError{
				HTTPCode: http.StatusConflict,
				Reason:   "Central cluster name is already used",
				Code:     36,
			},
		},
		{
			name: "does not throw an error when name is unique",
			arg: args{
				centralService: &services.CentralServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 0}, nil
					},
				},
				name:    "unique-name",
				context: context.TODO(),
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			validateFn := ValidateCentralClusterNameIsUnique(tt.arg.context, &tt.arg.name, tt.arg.centralService)
			err := validateFn()
			gomega.Expect(tt.want).To(gomega.Equal(err))
		})
	}
}

func Test_Validations_validateCentralClusterNames(t *testing.T) {
	tests := []struct {
		description string
		name        string
		expectError bool
	}{
		{
			description: "valid central cluster name",
			name:        "test-central1",
			expectError: false,
		},
		{
			description: "valid central cluster name with multiple '-'",
			name:        "test-my-cluster",
			expectError: false,
		},
		{
			description: "invalid central cluster name begins with number",
			name:        "1test-cluster",
			expectError: true,
		},
		{
			description: "invalid central cluster name with invalid characters",
			name:        "test-c%*_2",
			expectError: true,
		},
		{
			description: "invalid central cluster name with upper-case letters",
			name:        "Test-cluster",
			expectError: true,
		},
		{
			description: "invalid central cluster name with spaces",
			name:        "test cluster",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			validateFn := ValidCentralClusterName(&tt.name, "name")
			err := validateFn()
			if tt.expectError {
				gomega.Expect(err).Should(gomega.HaveOccurred())
			} else {
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}
		})
	}
}

func Test_Validation_validateCloudProvider(t *testing.T) {
	limit := int(5)
	evalMap := config.InstanceTypeMap{
		"eval": {
			Limit: &limit,
		},
	}
	standardMap := config.InstanceTypeMap{
		"standard": {
			Limit: &limit,
		},
	}
	type args struct {
		centralRequest dbapi.CentralRequest
		ProviderConfig *config.ProviderConfig
		centralService services.CentralService
	}

	type result struct {
		wantErr        bool
		reason         string
		centralRequest public.CentralRequest
	}

	tests := []struct {
		name string
		arg  args
		want result
	}{
		{
			name: "do not throw an error when default provider and region are picked",
			arg: args{
				centralService: &services.CentralServiceMock{
					DetectInstanceTypeFunc: func(centralRequest *dbapi.CentralRequest) (types.CentralInstanceType, *errors.ServiceError) {
						return types.EVAL, nil
					},
				},
				centralRequest: dbapi.CentralRequest{},
				ProviderConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name:    "aws",
								Default: true,
								Regions: config.RegionList{
									config.Region{
										Name:                   "us-east-1",
										Default:                true,
										SupportedInstanceTypes: evalMap,
									},
								},
							},
						},
					},
				},
			},
			want: result{
				wantErr: false,
				centralRequest: public.CentralRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
			},
		},
		{
			name: "do not throw an error when cloud provider and region matches",
			arg: args{
				centralService: &services.CentralServiceMock{
					DetectInstanceTypeFunc: func(centralRequest *dbapi.CentralRequest) (types.CentralInstanceType, *errors.ServiceError) {
						return types.EVAL, nil
					},
				},
				centralRequest: dbapi.CentralRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
				ProviderConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "gcp",
								Regions: config.RegionList{
									config.Region{
										Name:                   "eu-east-1",
										SupportedInstanceTypes: evalMap,
									},
								},
							},
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name:                   "us-east-1",
										SupportedInstanceTypes: evalMap,
									},
								},
							},
						},
					},
				},
			},
			want: result{
				wantErr: false,
				centralRequest: public.CentralRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
			},
		},
		{
			name: "throws an error when cloud provider and region do not match",
			arg: args{
				centralService: &services.CentralServiceMock{
					DetectInstanceTypeFunc: func(centralRequest *dbapi.CentralRequest) (types.CentralInstanceType, *errors.ServiceError) {
						return types.EVAL, nil
					},
				},
				centralRequest: dbapi.CentralRequest{
					CloudProvider: "aws",
					Region:        "us-east",
				},
				ProviderConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name:                   "us-east-1",
										SupportedInstanceTypes: evalMap,
									},
								},
							},
						},
					},
				},
			},
			want: result{
				wantErr: true,
				reason:  "region us-east is not supported for aws, supported regions are: [us-east-1]",
			},
		},
		{
			name: "throws an error when instance type is not supported",
			arg: args{
				centralService: &services.CentralServiceMock{
					DetectInstanceTypeFunc: func(centralRequest *dbapi.CentralRequest) (types.CentralInstanceType, *errors.ServiceError) {
						return types.EVAL, nil
					},
				},
				centralRequest: dbapi.CentralRequest{
					CloudProvider: "aws",
					Region:        "us-east",
				},
				ProviderConfig: &config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name:                   "us-east",
										SupportedInstanceTypes: standardMap,
									},
								},
							},
						},
					},
				},
			},
			want: result{
				wantErr: true,
				reason:  "instance type 'eval' not supported for region 'us-east'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			validateFn := ValidateCloudProvider(&tt.arg.centralService, &tt.arg.centralRequest, tt.arg.ProviderConfig, "creating-central")
			err := validateFn()
			if !tt.want.wantErr && err != nil {
				t.Errorf("validatedCloudProvider() expected not to throw error but threw %v", err)
			} else if tt.want.wantErr {
				gomega.Expect(err.Reason).To(gomega.Equal(tt.want.reason))
				return
			}

			gomega.Expect(tt.want.wantErr).To(gomega.Equal(err != nil))

			if !tt.want.wantErr {
				gomega.Expect(tt.arg.centralRequest.CloudProvider).To(gomega.Equal(tt.want.centralRequest.CloudProvider))
				gomega.Expect(tt.arg.centralRequest.Region).To(gomega.Equal(tt.want.centralRequest.Region))
			}

		})
	}
}
