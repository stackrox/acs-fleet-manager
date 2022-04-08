package clusters

import (
	"reflect"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/config"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	mockClusterMultiAZ       = true
	mockClusterCloudProvider = "aws"
	mockClusterRegion        = "us-east-1"
	mockClusterBYOC          = false
	mockClusterManaged       = true
	openshiftVersion         = "openshift-v4.6.1"
)

func Test_clusterBuilder_NewOCMClusterFromCluster(t *testing.T) {
	awsConfig := &config.AWSConfig{}

	dataplaneClusterConfig := &config.DataplaneClusterConfig{
		OpenshiftVersion:   openshiftVersion,
		ComputeMachineType: ComputeMachineType,
	}

	clusterAWS := clustersmgmtv1.
		NewAWS().
		AccountID(awsConfig.AccountID).
		AccessKeyID(awsConfig.AccessKey).
		SecretAccessKey(awsConfig.SecretAccessKey)

	type fields struct {
		idGenerator            ocm.IDGenerator
		awsConfig              *config.AWSConfig
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	type args struct {
		clusterRequest *types.ClusterRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantFn  func() *clustersmgmtv1.Cluster
		wantErr bool
	}{
		{
			name: "nil aws config results in error",
			fields: fields{
				idGenerator:            ocm.NewIDGenerator(""),
				awsConfig:              nil,
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{},
			},
			wantErr: true,
		},
		{
			name: "nil cluster creation config results in error",
			fields: fields{
				idGenerator:            ocm.NewIDGenerator(""),
				awsConfig:              awsConfig,
				dataplaneClusterConfig: nil,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{},
			},
			wantErr: true,
		},
		{
			name: "nil cluster results in error",
			fields: fields{
				idGenerator:            ocm.NewIDGenerator(""),
				awsConfig:              awsConfig,
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: nil,
			},
			wantErr: true,
		},
		{
			name: "nil id generator results in error",
			fields: fields{
				idGenerator:            nil,
				awsConfig:              awsConfig,
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{},
			},
			wantErr: true,
		},
		{
			name: "successful conversion of all supported provided values",
			fields: fields{
				idGenerator: &ocm.IDGeneratorMock{
					GenerateFunc: func() string {
						return ""
					},
				},
				awsConfig:              awsConfig,
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{
					CloudProvider: mockClusterCloudProvider,
					Region:        mockClusterRegion,
					MultiAZ:       mockClusterMultiAZ,
				},
			},
			wantFn: func() *clustersmgmtv1.Cluster {
				cluster, err := newMockCluster(func(builder *clustersmgmtv1.ClusterBuilder) {
					// these values will be ignored by the conversion as they're unsupported. so expect different
					// values than we provide.
					builder.CCS(clustersmgmtv1.NewCCS().Enabled(true))
					builder.Managed(true)
					builder.Name("")
					builder.AWS(clusterAWS)
					builder.MultiAZ(true)
					builder.Version(clustersmgmtv1.NewVersion().ID(openshiftVersion))
					builder.Nodes(clustersmgmtv1.NewClusterNodes().ComputeMachineType(clustersmgmtv1.NewMachineType().ID(ComputeMachineType)))
				})
				if err != nil {
					panic(err)
				}
				return cluster
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if tt.wantFn == nil {
			tt.wantFn = func() *clustersmgmtv1.Cluster {
				return nil
			}
		}
		t.Run(tt.name, func(t *testing.T) {
			r := clusterBuilder{
				idGenerator:            tt.fields.idGenerator,
				awsConfig:              tt.fields.awsConfig,
				dataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
			}
			got, err := r.NewOCMClusterFromCluster(tt.args.clusterRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOCMClusterFromCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantFn()) {
				t.Errorf("NewOCMClusterFromCluster() got = %+v, want %+v", got, tt.wantFn())
			}
		})
	}
}

// newMockCluster create a default OCM Cluster Service cluster struct and apply modifications if provided.
func newMockCluster(modifyFn func(*clustersmgmtv1.ClusterBuilder)) (*clustersmgmtv1.Cluster, error) {
	mock := clustersmgmtv1.NewCluster()
	mock.CloudProvider(clustersmgmtv1.NewCloudProvider().ID(mockClusterCloudProvider))
	mock.Region(clustersmgmtv1.NewCloudRegion().ID(mockClusterRegion))
	mock.CCS(clustersmgmtv1.NewCCS().Enabled(mockClusterBYOC))
	mock.Managed(mockClusterManaged)
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock.Build()
}
