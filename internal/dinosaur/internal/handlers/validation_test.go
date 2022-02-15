package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/dinosaurs/types"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
	"github.com/onsi/gomega"
)

func Test_Validation_validateDinosaurClusterNameIsUnique(t *testing.T) {
	type args struct {
		dinosaurService services.DinosaurService
		name            string
		context         context.Context
	}

	tests := []struct {
		name string
		arg  args
		want *errors.ServiceError
	}{
		{
			name: "throw an error when the DinosaurService call throws an error",
			arg: args{
				dinosaurService: &services.DinosaurServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (dbapi.DinosaurList, *api.PagingMeta, *errors.ServiceError) {
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
				dinosaurService: &services.DinosaurServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (dbapi.DinosaurList, *api.PagingMeta, *errors.ServiceError) {
						return nil, &api.PagingMeta{Total: 1}, nil
					},
				},
				name:    "duplicate-name",
				context: context.TODO(),
			},
			want: &errors.ServiceError{
				HttpCode: http.StatusConflict,
				Reason:   "Dinosaur cluster name is already used",
				Code:     36,
			},
		},
		{
			name: "does not throw an error when name is unique",
			arg: args{
				dinosaurService: &services.DinosaurServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreServices.ListArguments) (dbapi.DinosaurList, *api.PagingMeta, *errors.ServiceError) {
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
			validateFn := ValidateDinosaurClusterNameIsUnique(&tt.arg.name, tt.arg.dinosaurService, tt.arg.context)
			err := validateFn()
			gomega.Expect(tt.want).To(gomega.Equal(err))
		})
	}
}

func Test_Validations_validateDinosaurClusterNames(t *testing.T) {
	tests := []struct {
		description string
		name        string
		expectError bool
	}{
		{
			description: "valid dinosaur cluster name",
			name:        "test-dinosaur1",
			expectError: false,
		},
		{
			description: "valid dinosaur cluster name with multiple '-'",
			name:        "test-my-cluster",
			expectError: false,
		},
		{
			description: "invalid dinosaur cluster name begins with number",
			name:        "1test-cluster",
			expectError: true,
		},
		{
			description: "invalid dinosaur cluster name with invalid characters",
			name:        "test-c%*_2",
			expectError: true,
		},
		{
			description: "invalid dinosaur cluster name with upper-case letters",
			name:        "Test-cluster",
			expectError: true,
		},
		{
			description: "invalid dinosaur cluster name with spaces",
			name:        "test cluster",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			validateFn := ValidDinosaurClusterName(&tt.name, "name")
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
		dinosaurRequest dbapi.DinosaurRequest
		ProviderConfig  *config.ProviderConfig
		dinosaurService services.DinosaurService
	}

	type result struct {
		wantErr         bool
		reason          string
		dinosaurRequest public.DinosaurRequest
	}

	tests := []struct {
		name string
		arg  args
		want result
	}{
		{
			name: "do not throw an error when default provider and region are picked",
			arg: args{
				dinosaurService: &services.DinosaurServiceMock{
					DetectInstanceTypeFunc: func(dinosaurRequest *dbapi.DinosaurRequest) (types.DinosaurInstanceType, *errors.ServiceError) {
						return types.EVAL, nil
					},
				},
				dinosaurRequest: dbapi.DinosaurRequest{},
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
				dinosaurRequest: public.DinosaurRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
			},
		},
		{
			name: "do not throw an error when cloud provider and region matches",
			arg: args{
				dinosaurService: &services.DinosaurServiceMock{
					DetectInstanceTypeFunc: func(dinosaurRequest *dbapi.DinosaurRequest) (types.DinosaurInstanceType, *errors.ServiceError) {
						return types.EVAL, nil
					},
				},
				dinosaurRequest: dbapi.DinosaurRequest{
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
				dinosaurRequest: public.DinosaurRequest{
					CloudProvider: "aws",
					Region:        "us-east-1",
				},
			},
		},
		{
			name: "throws an error when cloud provider and region do not match",
			arg: args{
				dinosaurService: &services.DinosaurServiceMock{
					DetectInstanceTypeFunc: func(dinosaurRequest *dbapi.DinosaurRequest) (types.DinosaurInstanceType, *errors.ServiceError) {
						return types.EVAL, nil
					},
				},
				dinosaurRequest: dbapi.DinosaurRequest{
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
				dinosaurService: &services.DinosaurServiceMock{
					DetectInstanceTypeFunc: func(dinosaurRequest *dbapi.DinosaurRequest) (types.DinosaurInstanceType, *errors.ServiceError) {
						return types.EVAL, nil
					},
				},
				dinosaurRequest: dbapi.DinosaurRequest{
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
			validateFn := ValidateCloudProvider(&tt.arg.dinosaurService, &tt.arg.dinosaurRequest, tt.arg.ProviderConfig, "creating-dinosaur")
			err := validateFn()
			if !tt.want.wantErr && err != nil {
				t.Errorf("validatedCloudProvider() expected not to throw error but threw %v", err)
			} else if tt.want.wantErr {
				gomega.Expect(err.Reason).To(gomega.Equal(tt.want.reason))
				return
			}

			gomega.Expect(tt.want.wantErr).To(gomega.Equal(err != nil))

			if !tt.want.wantErr {
				gomega.Expect(tt.arg.dinosaurRequest.CloudProvider).To(gomega.Equal(tt.want.dinosaurRequest.CloudProvider))
				gomega.Expect(tt.arg.dinosaurRequest.Region).To(gomega.Equal(tt.want.dinosaurRequest.Region))
			}

		})
	}
}
