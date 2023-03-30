package services

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/converters"

	"github.com/pkg/errors"

	apiErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"

	"github.com/onsi/gomega"
	"gorm.io/gorm"

	mocket "github.com/selvatico/go-mocket"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

var (
	testRegion   = "us-west-1"
	testProvider = "aws"
	testDNS      = "apps.mk-btq2d1h8d3b1.b3k3.s1.devshift.org"
	testMultiAZ  = true
	testStatus   = api.ClusterProvisioned
)

// build a test cluster
func buildCluster(modifyFn func(cluster *api.Cluster)) *api.Cluster {
	cluster := &api.Cluster{
		Region:        testRegion,
		CloudProvider: testProvider,
		MultiAZ:       testMultiAZ,
		ProviderType:  api.ClusterProviderOCM,
		Meta: api.Meta{
			DeletedAt: gorm.DeletedAt{Valid: true},
		},
		Schedulable: true,
	}
	if modifyFn != nil {
		modifyFn(cluster)
	}
	return cluster
}

func checkClusterFields(this *api.Cluster, that *api.Cluster) bool {
	if this == that {
		return true
	}
	if this.ClusterID != that.ClusterID || this.ExternalID != that.ExternalID || this.Region != that.Region || this.MultiAZ != that.MultiAZ || this.ProviderType != that.ProviderType || this.Status != that.Status || this.CloudProvider != that.CloudProvider {
		return false
	}
	return true
}

func Test_Cluster_Create(t *testing.T) {
	testClusterInternalID := "test-cluster-id"
	testClusterExternalID := "test-cluster-external-id"
	wantedCluster := &api.Cluster{
		CloudProvider: testProvider,
		ClusterID:     testClusterInternalID,
		ExternalID:    testClusterExternalID,
		MultiAZ:       testMultiAZ,
		Region:        testRegion,
		Status:        api.ClusterProvisioning,
		ProviderType:  api.ClusterProviderOCM,
		ProviderSpec:  nil,
		ClusterSpec:   nil,
		Schedulable:   true,
	}

	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster *api.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    *api.Cluster
	}{
		{
			name: "successful cluster creation from cluster request job",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						CreateFunc: func(request *types.ClusterRequest) (*types.ClusterSpec, error) {
							return &types.ClusterSpec{
								InternalID: testClusterInternalID,
								ExternalID: testClusterExternalID,
								Status:     api.ClusterProvisioning,
							}, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: buildCluster(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`INSERT INTO "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
			want:    wantedCluster,
		},
		{
			name: "CreateCluster failure",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						CreateFunc: func(request *types.ClusterRequest) (*types.ClusterSpec, error) {
							return nil, errors.New("CreateCluster failure")
						},
					}, nil
				}},
			},
			args: args{
				cluster: buildCluster(nil),
			},
			wantErr: true,
		},
		{
			name: "Database error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						CreateFunc: func(request *types.ClusterRequest) (*types.ClusterSpec, error) {
							return &types.ClusterSpec{
								InternalID: testClusterInternalID,
								ExternalID: testClusterExternalID,
								Status:     api.ClusterProvisioning,
							}, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: buildCluster(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("INSERT").WithError(fmt.Errorf("database error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.Create(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !checkClusterFields(got, tt.want) {
				t.Errorf("Create() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_GetClusterDNS(t *testing.T) {
	mockClusterDNS := testDNS

	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		clusterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    string
	}{
		{
			name: "successful retrieval of clusterDNS",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{GetClusterDNSFunc: func(clusterSpec *types.ClusterSpec) (string, error) {
						return mockClusterDNS, nil
					}}, nil
				}},
			},
			args: args{
				clusterID: testClusterID,
			},
			setupFn: func() {
				res := []map[string]interface{}{
					{
						"id":          "testid",
						"cluster_id":  "testid",
						"cluster_dns": "",
					},
				}
				mocket.Catcher.Reset().
					NewMock().WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithReply(res)
			},
			wantErr: false,
			want:    mockClusterDNS,
		},
		{
			name: "error when passing empty clusterID",
			fields: fields{
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{GetClusterDNSFunc: func(clusterSpec *types.ClusterSpec) (string, error) {
						return mockClusterDNS, nil
					}}, nil
				}},
			},
			args: args{
				clusterID: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.GetClusterDNS(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClusterDNS() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if got == "" && !tt.wantErr {
				t.Errorf("GetClusterDNS() error - expecting non-empty cluster DNS here, got '%v'", got)
			}
		})
	}
}

func Test_Cluster_FindClusterByID(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.Cluster
		wantErr bool
		setupFn func()
	}{
		{
			name: "nil and no error when id is not found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id: "non-existentID",
			},
			wantErr: false,
			want:    nil,
		},
		{
			name: "error when id is empty (undefined)",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id: "",
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "error when sql where query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id: testClusterID,
			},
			wantErr: true,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "successful output",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				id: testClusterID,
			},
			want: &api.Cluster{ClusterID: testClusterID},
			setupFn: func() {
				mockedResponse := []map[string]interface{}{{"cluster_id": testClusterID}}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithReply(mockedResponse)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.FindClusterByID(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindClusterByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindClusterByID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_FindCluster(t *testing.T) {
	clusterDetails := FindClusterCriteria{
		Provider: testProvider,
		Region:   testRegion,
		MultiAZ:  testMultiAZ,
		Status:   testStatus,
	}

	nonExistentClusterDetails := FindClusterCriteria{
		Provider: "nonExistentProvider",
		Region:   testRegion,
		MultiAZ:  testMultiAZ,
		Status:   testStatus,
	}

	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		criteria FindClusterCriteria
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    *api.Cluster
	}{
		{
			name: "return nil if no cluster is found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				criteria: nonExistentClusterDetails,
			},
			want:    nil,
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
		{
			name: "error when sql where query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				criteria: clusterDetails,
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "successful retrieval of a cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				criteria: clusterDetails,
			},
			want: buildCluster(nil),
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`).WithReply(converters.ConvertCluster(buildCluster(nil)))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}

			got, err := c.FindCluster(tt.args.criteria)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindCluster() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindCluster() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_ListByStatus(t *testing.T) {
	var nonEmptyClusterList = []api.Cluster{
		{
			CloudProvider: testProvider,
			MultiAZ:       testMultiAZ,
			Region:        testRegion,
			Status:        testStatus,
			Meta: api.Meta{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				DeletedAt: gorm.DeletedAt{Valid: true},
			},
		},
	}

	var emptyClusterList []api.Cluster

	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		status api.ClusterStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []api.Cluster
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when status is undefined",
			args: args{
				status: "",
			},
			wantErr: true,
		},
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status: testStatus,
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "success: return empty list of clusters with specified status",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status: testStatus,
			},
			want:    emptyClusterList,
			wantErr: false,
			setupFn: func() {
				response := converters.ConvertClusterList(emptyClusterList)
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`).WithReply(response)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "success: return non-empty list of clusters with specified status",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status: testStatus,
			},
			want:    nonEmptyClusterList,
			wantErr: false,
			setupFn: func() {
				response := converters.ConvertClusterList(nonEmptyClusterList)
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE status = $1`).
					WithArgs(api.ClusterProvisioned.String()).
					WithReply(response)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.ListByStatus(tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListByStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gomega.Expect(got).To(gomega.Equal(tt.want))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListByStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ClusterService_Update(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		cluster api.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    error
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when id is undefined",
			args: args{
				cluster: api.Cluster{},
			},
			wantErr: true,
		},
		{
			name: "error when database update returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithExecException()
			},
		},
		{
			name: "successful status update by id",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				cluster: api.Cluster{Meta: api.Meta{ID: testID}},
			},
			wantErr: false,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().
					WithQuery(`UPDATE "clusters" SET "id"=$1,"updated_at"=$2 WHERE "clusters"."deleted_at" IS NULL AND "id" = $3`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.Update(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_UpdateStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		cluster api.Cluster
		status  api.ClusterStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    error
		wantErr bool
		setupFn func()
	}{
		{
			name: "error when status is undefined",
			args: args{
				status: "",
			},
			wantErr: true,
		},
		{
			name: "error when id is undefined",
			args: args{
				cluster: api.Cluster{},
				status:  testStatus,
			},
			wantErr: true,
		},
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithExecException()
			},
		},
		{
			name: "successful status update by id",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status:  testStatus,
				cluster: api.Cluster{Meta: api.Meta{ID: testID}},
			},
			wantErr: false,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters" SET "status"=$1,"updated_at"=$2 WHERE id = $3`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "successful status update by ClusterID",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status:  testStatus,
				cluster: api.Cluster{ClusterID: testID},
			},
			wantErr: false,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters" SET "status"=$1,"updated_at"=$2 WHERE cluster_id = $3`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.UpdateStatus(tt.args.cluster, tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_RegisterClusterJob(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		clusterRequest api.Cluster
		status         api.ClusterStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    error
		wantErr bool
		setupFn func()
	}{
		{
			name: "success registering a new job",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				status: "",
				clusterRequest: api.Cluster{
					CloudProvider: "",
					ClusterID:     "",
					ExternalID:    "",
					MultiAZ:       false,
					Region:        "",
				},
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`INSERT INTO "clusters"`)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}

			err := k.RegisterClusterJob(&tt.args.clusterRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterClusterJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_ScaleUpComputeNodes(t *testing.T) {
	testNodeIncrement := 3
	type fields struct {
		clusterProviderFactory clusters.ProviderFactory
		connectionFactory      *db.ConnectionFactory
	}
	type args struct {
		clusterID string
	}

	type checkError struct {
		wantErr bool
		error   string
	}

	tests := []struct {
		name       string
		fields     fields
		args       args
		checkError checkError
		setupFn    func()
	}{
		{
			name: "error when cluster id is undefined",
			args: args{
				clusterID: "",
			},
			checkError: checkError{
				wantErr: true,
			},
		},
		{
			name: "error when scale up compute nodes ocm function fails",
			args: args{
				clusterID: testID,
			},
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{ScaleUpFunc: func(clusterSpec *types.ClusterSpec, increment int) (*types.ClusterSpec, error) {
						return nil, errors.New("test ScaleUpComputeNodes failure")
					}}, nil
				}},
			},
			setupFn: func() {
				res := []map[string]interface{}{
					{
						"id":          "testid",
						"cluster_id":  "testid",
						"cluster_dns": "",
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testID).
					WithReply(res)
			},
			checkError: checkError{
				wantErr: true,
				error:   "RHACS-MGMT-9: failed to scale up cluster",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				providerFactory:   tt.fields.clusterProviderFactory,
				connectionFactory: tt.fields.connectionFactory,
			}
			_, err := k.ScaleUpComputeNodes(tt.args.clusterID, testNodeIncrement)
			if (err != nil) != tt.checkError.wantErr {
				t.Errorf("ScaleUpComputeNodes() error = %v, wantErr %v", err, tt.checkError.wantErr)
				return
			}
			if err != nil && !strings.Contains(err.Error(), tt.checkError.error) {
				t.Errorf("Wrong error received. Expecting '%s' but received '%s'", tt.checkError.error, err.Error())
			}
		})
	}
}

func Test_ScaleDownComputeNodes(t *testing.T) {
	testNodeDecrement := 3
	type fields struct {
		clusterProviderFactory clusters.ProviderFactory
		connectionFactory      *db.ConnectionFactory
	}
	type args struct {
		clusterID string
	}
	type errorCheck struct {
		wantErr   bool
		expectErr string
	}

	tests := []struct {
		name       string
		fields     fields
		args       args
		errorCheck errorCheck
		setupFn    func()
	}{
		{
			name: "error when cluster id is undefined",
			args: args{
				clusterID: "",
			},
			errorCheck: errorCheck{
				wantErr: true,
			},
		},
		{
			name: "error when scale down ocm function fails",
			args: args{
				clusterID: testID,
			},
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{ScaleDownFunc: func(clusterSpec *types.ClusterSpec, increment int) (*types.ClusterSpec, error) {
						return nil, errors.New("test ScaleDownComputeNodes failure")
					}}, nil
				}},
			},
			setupFn: func() {
				res := []map[string]interface{}{
					{
						"id":          "testid",
						"cluster_id":  "testid",
						"cluster_dns": "",
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testID).
					WithReply(res)
			},
			errorCheck: errorCheck{
				wantErr:   true,
				expectErr: "RHACS-MGMT-9: failed to scale down cluster",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				providerFactory:   tt.fields.clusterProviderFactory,
				connectionFactory: tt.fields.connectionFactory,
			}
			_, err := k.ScaleDownComputeNodes(tt.args.clusterID, testNodeDecrement)
			if (err != nil) != tt.errorCheck.wantErr {
				t.Errorf("ScaleDownComputeNodes() error = %v, wantErr %v", err, tt.errorCheck.wantErr)
				return
			}
			if err != nil && !strings.Contains(err.Error(), tt.errorCheck.expectErr) {
				t.Errorf("Wrong error received. Expecting '%s' but received '%s'", tt.errorCheck.expectErr, err.Error())
			}
		})
	}
}

func TestClusterService_ListGroupByProviderAndRegion(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
		providers         []string
		regions           []string
		status            []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    []*ResGroupCPRegion
		setupFn func()
	}{
		{
			name: "ListGroupByProviderAndRegion success",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providers:         []string{"aws"},
				regions:           []string{"us-east-1"},
				status:            api.StatusForValidCluster,
			},
			want: []*ResGroupCPRegion{
				{
					Provider: "aws",
					Region:   "east-1",
					Count:    1,
				},
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`FROM "clusters"`).WithReply([]map[string]interface{}{
					{
						"Provider": "aws",
						"Region":   "east-1",
						"Count":    1,
					},
				})
			},
		},
		{
			name: "ListGroupByProviderAndRegion failure",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				providers:         []string{"aws"},
				regions:           []string{"us-east-1"},
				status:            api.StatusForValidCluster,
			},
			want:    nil,
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.ListGroupByProviderAndRegion(tt.fields.providers, tt.fields.regions, tt.fields.status)
			if err != nil && !tt.wantErr {
				t.Errorf("ListGroupByProviderAndRegion err = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListGroupByProviderAndRegion got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_DeleteByClusterId(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		clusterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    error
		wantErr bool
		setupFn func()
	}{
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithError(fmt.Errorf("database error"))
			},
		},
		{
			name: "successful soft delete cluster by id",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterID: "123",
			},
			wantErr: false,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			err := k.DeleteByClusterID(tt.args.clusterID)
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_Cluster_FindNonEmptyClusterByID(t *testing.T) {
	now := time.Now()

	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		clusterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.Cluster
		wantErr bool
		setupFn func()
	}{
		{
			name: "nil and no error when id is not found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterID: "non-existentID",
			},
			wantErr: false,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
		{
			name: "error when sql where query fails",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterID: testClusterID,
			},
			wantErr: true,
			want:    nil,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithError(fmt.Errorf("some database error"))
			},
		},
		{
			name: "successful find the cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterID: testClusterID,
			},
			want: &api.Cluster{ClusterID: testClusterID, Meta: api.Meta{CreatedAt: now, UpdatedAt: now, DeletedAt: gorm.DeletedAt{Valid: true}}},
			setupFn: func() {
				mockedResponse := []map[string]interface{}{{"cluster_id": testClusterID, "created_at": now, "updated_at": now, "deleted_at": gorm.DeletedAt{Valid: true}.Time}}
				query := `SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1 AND cluster_id IN (SELECT "cluster_id" FROM "central_requests" WHERE (status != $2 AND cluster_id = $3) AND "central_requests"."deleted_at" IS NULL) AND "clusters"."deleted_at" IS NULL ORDER BY "clusters"."id" LIMIT 1%`
				mocket.Catcher.Reset().NewMock().WithQuery(query).WithReply(mockedResponse)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}
			k := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := k.FindNonEmptyClusterByID(tt.args.clusterID)
			gomega.Expect(got).To(gomega.Equal(tt.want))
			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_clusterService_ListAllClusterIds(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	var clusters []api.Cluster
	clusters = append(clusters, api.Cluster{ClusterID: "test01"})

	tests := []struct {
		name    string
		fields  fields
		want    []api.Cluster
		setupFn func()
		want1   *apiErrors.ServiceError
	}{
		{
			name: "Empty cluster Ids",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT "cluster_id" FROM "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			want:  nil,
			want1: nil,
		},
		{
			name: "List All cluster id",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT "cluster_id" FROM "clusters" WHERE cluster_id != ''`).WithReply([]map[string]interface{}{
					{
						"cluster_id": "test01",
					},
				})
			},
			want:  clusters,
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, got1 := c.ListAllClusterIds()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListAllClusterIds() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) { //nolint:govet
				t.Errorf("ListAllClusterIds() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_clusterService_FindDinosaurInstanceCount(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		clusterID []string
	}
	var testRes []ResDinosaurInstanceCount
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []ResDinosaurInstanceCount
		wantErr bool
		setupFn func()
	}{
		{
			name: "Instance count equals to 2",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				[]string{"test01", "test02"},
			},
			want: []ResDinosaurInstanceCount{
				{
					Clusterid: "test01",
					Count:     2,
				},
				{
					Clusterid: "test02",
					Count:     0,
				},
			},
			wantErr: false,
			setupFn: func() {
				counters := []map[string]interface{}{
					{
						"clusterid": "test01",
						"count":     2,
					},
				}
				mocket.Catcher.Reset().NewMock().WithQuery(`GROUP BY "cluster_id"`).WithReply(counters)
			},
		},
		{
			name: "Instance count with exception",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				[]string{},
			},
			want:    testRes,
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithQueryException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := c.FindDinosaurInstanceCount(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindDinosaurInstanceCount() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			for i, res := range got {
				if !reflect.DeepEqual(res, tt.want[i]) {
					t.Errorf("FindDinosaurInstanceCount() got = %v, want %v", res, tt.want[i])
				}
			}
		})
	}
}

func Test_clusterService_FindAllClusters(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	clusterReady := FindClusterCriteria{
		Provider: "test-provider",
		Region:   "us-east",
		MultiAZ:  true,
		Status:   api.ClusterReady,
	}
	var clusters []*api.Cluster
	clusters = append(clusters, &api.Cluster{ClusterID: "test01", Status: api.ClusterReady, Meta: api.Meta{
		DeletedAt: gorm.DeletedAt{
			Valid: true,
		},
	}})

	type args struct {
		criteria FindClusterCriteria
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*api.Cluster
		setupFn func()
		wantErr bool
	}{
		{
			name: "Find all cluster with empty result",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				criteria: clusterReady,
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
		{
			name: "Find all clusters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT * FROM "clusters"`).WithReply(converters.ConvertClusters(clusters))
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			args: args{
				criteria: clusterReady,
			},
			want:    clusters,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			got, err := c.FindAllClusters(tt.args.criteria)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindAllClusters() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			for i, res := range got {
				if !reflect.DeepEqual(*res, *tt.want[i]) {
					t.Errorf("FindAllClusters() got = %v, want %v", *res, *tt.want[i])
				}
			}

		})
	}
}

func Test_clusterService_UpdateMultiClusterStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		clusterIds []string
		status     api.ClusterStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setupFn func()
	}{
		{
			name: "nil and no error when id is not found",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterIds: []string{"notexists"},
				status:     api.ClusterDeprovisioning,
			},
			wantErr: false,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
		{
			name: "error when ids is undefined",
			args: args{
				clusterIds: []string{},
				status:     api.ClusterDeprovisioning,
			},
			wantErr: true,
		},
		{
			name: "error when status is undefined",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterIds: []string{"notexists"},
			},
			wantErr: true,
		},
		{
			name: "fail: database returns an error",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				clusterIds: []string{"notexists"},
				status:     api.ClusterDeprovisioning,
			},
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("UPDATE").WithExecException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			if err := c.UpdateMultiClusterStatus(tt.args.clusterIds, tt.args.status); (err != nil) != tt.wantErr {
				t.Errorf("UpdateMultiClusterStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClusterService_CountByStatus(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		status []api.ClusterStatus
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		want      []ClusterStatusCount
		setupFunc func()
	}{
		{
			name:   "should return the counts of clusters in different status",
			fields: fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			args: args{
				status: []api.ClusterStatus{api.ClusterAccepted, api.ClusterReady, api.ClusterProvisioning},
			},
			wantErr: false,
			setupFunc: func() {
				counters := []map[string]interface{}{
					{
						"status": "cluster_accepted",
						"count":  2,
					},
					{
						"status": "ready",
						"count":  1,
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT status as Status, count(1) as Count FROM "clusters" WHERE status in ($1,$2,$3)`).
					WithArgs(api.ClusterAccepted.String(), api.ClusterReady.String(), api.ClusterProvisioning.String()).
					WithReply(counters)
			},
			want: []ClusterStatusCount{{
				Status: api.ClusterAccepted,
				Count:  2,
			}, {
				Status: api.ClusterReady,
				Count:  1,
			}, {
				Status: api.ClusterProvisioning,
				Count:  0,
			}},
		},
		{
			name:   "should return error",
			fields: fields{connectionFactory: db.NewMockConnectionFactory(nil)},
			args: args{
				status: []api.ClusterStatus{api.ClusterAccepted, api.ClusterReady},
			},
			wantErr: true,
			setupFunc: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`SELECT`).WithQueryException()
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			c := clusterService{
				connectionFactory: tt.fields.connectionFactory,
			}
			status, err := c.CountByStatus(tt.args.status)
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for CountByStatus: %v", err)
			}
			if !reflect.DeepEqual(status, tt.want) {
				t.Errorf("CountByStatus want = %v, got = %v", tt.want, status)
			}
		})
	}
}

func TestClusterService_GetComputeNodes(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		clusterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    *types.ComputeNodesInfo
	}{
		{
			name: "successful get compute nodes info",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						GetComputeNodesFunc: func(spec *types.ClusterSpec) (*types.ComputeNodesInfo, error) {
							return &types.ComputeNodesInfo{
								Actual:  3,
								Desired: 3,
							}, nil
						},
					}, nil
				}},
			},
			args: args{
				clusterID: testClusterID,
			},
			setupFn: func() {
				res := []map[string]interface{}{
					{
						"id":          testClusterID,
						"cluster_id":  testClusterID,
						"cluster_dns": "",
					},
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithReply(res)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
			want: &types.ComputeNodesInfo{
				Actual:  3,
				Desired: 3,
			},
		},
		{
			name: "error when passing empty clusterID",
			fields: fields{
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return nil, errors.Errorf("this function should not called")
				}},
			},
			args: args{
				clusterID: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.GetComputeNodes(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetComputeNodes() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetComputeNodes() want %v but got %v", tt.want, got)
			}
		})
	}
}

func TestClusterService_CheckClusterStatus(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster *api.Cluster
	}

	clusterID := "test-internal-id"
	clusterExternalID := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    *api.Cluster
	}{
		{
			name: "successfully check cluster status",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						CheckClusterStatusFunc: func(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
							return &types.ClusterSpec{
								InternalID: clusterID,
								ExternalID: clusterExternalID,
								Status:     api.ClusterProvisioned,
							}, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: &api.Cluster{
					Meta: api.Meta{
						ID: clusterID,
					},
					ExternalID: clusterExternalID,
					ClusterID:  clusterID,
					Status:     clusterStatus,
				},
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters" SET "id"=$1`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
			want: &api.Cluster{
				Meta:       api.Meta{ID: clusterID},
				ClusterID:  clusterID,
				ExternalID: clusterExternalID,
				Status:     api.ClusterProvisioned,
			},
		},
		{
			name: "error when failed to check cluster status",
			fields: fields{
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						CheckClusterStatusFunc: func(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
							return nil, errors.Errorf("failed to get cluster status")
						},
					}, nil
				}},
			},
			args: args{
				cluster: &api.Cluster{
					Meta: api.Meta{
						ID: clusterID,
					},
					ExternalID: clusterExternalID,
					ClusterID:  clusterID,
					Status:     clusterStatus,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.CheckClusterStatus(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckClusterStatus() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckClusterStatus() want %v but got %v", tt.want, got)
			}
		})
	}
}

func TestClusterService_RemoveClusterFromProvider(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster *api.Cluster
	}

	clusterID := "test-internal-id"
	clusterExternalID := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterID,
		},
		ExternalID: clusterExternalID,
		ClusterID:  clusterID,
		Status:     clusterStatus,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    bool
	}{
		{
			name: "successfully delete cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						DeleteFunc: func(spec *types.ClusterSpec) (bool, error) {
							return true, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: cluster,
			},
			wantErr: false,
			want:    true,
		},
		{
			name: "error when failed to delete cluster",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						DeleteFunc: func(spec *types.ClusterSpec) (bool, error) {
							return false, errors.Errorf("failed to delete cluster")
						},
					}, nil
				}},
			},
			args: args{
				cluster: cluster,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.Delete(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Delete() want %v but got %v", tt.want, got)
			}
		})
	}
}

func TestClusterService_ConfigureAndSaveIdentityProvider(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster          *api.Cluster
		identityProvider types.IdentityProviderInfo
	}

	clusterID := "test-internal-id"
	clusterExternalID := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterID,
		},
		ExternalID: clusterExternalID,
		ClusterID:  clusterID,
		Status:     clusterStatus,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    *api.Cluster
	}{
		{
			name: "successfully configured identity provider",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						AddIdentityProviderFunc: func(clusterSpec *types.ClusterSpec, identityProvider types.IdentityProviderInfo) (*types.IdentityProviderInfo, error) {
							return &types.IdentityProviderInfo{
								OpenID: &types.OpenIDIdentityProviderInfo{
									ID: "test-id",
								},
							}, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: &api.Cluster{
					Meta: api.Meta{
						ID: clusterID,
					},
					ExternalID: clusterExternalID,
					ClusterID:  clusterID,
					Status:     clusterStatus,
				},
				identityProvider: types.IdentityProviderInfo{OpenID: &types.OpenIDIdentityProviderInfo{
					Name:         "test-name",
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret", // pragma: allowlist secret
					Issuer:       "test-issuer",
				}},
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery(`UPDATE "clusters"`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
			wantErr: false,
			want: &api.Cluster{
				Meta: api.Meta{
					ID: clusterID,
				},
				ExternalID:         clusterExternalID,
				ClusterID:          clusterID,
				Status:             clusterStatus,
				IdentityProviderID: "test-id",
			},
		},
		{
			name: "error when failed to add identity provider",
			fields: fields{
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						AddIdentityProviderFunc: func(clusterSpec *types.ClusterSpec, identityProvider types.IdentityProviderInfo) (*types.IdentityProviderInfo, error) {
							return nil, errors.Errorf("failed to add identity provider")
						},
					}, nil
				}},
			},
			args: args{
				cluster: cluster,
				identityProvider: types.IdentityProviderInfo{OpenID: &types.OpenIDIdentityProviderInfo{
					Name:         "test-name",
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret", // pragma: allowlist secret
					Issuer:       "test-issuer",
				}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.ConfigureAndSaveIdentityProvider(tt.args.cluster, tt.args.identityProvider)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigureAndSaveIdentityProvider() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConfigureAndSaveIdentityProvider() want %v but got %v", tt.want, got)
			}
		})
	}
}

func TestClusterService_ApplyResources(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster   *api.Cluster
		resources types.ResourceSet
	}

	clusterID := "test-internal-id"
	clusterExternalID := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterID,
		},
		ExternalID: clusterExternalID,
		ClusterID:  clusterID,
		Status:     clusterStatus,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
	}{
		{
			name: "successfully applied resources",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						ApplyResourcesFunc: func(clusterSpec *types.ClusterSpec, resources types.ResourceSet) (*types.ResourceSet, error) {
							return nil, nil
						},
					}, nil
				}},
			},
			args: args{
				cluster: cluster,
				resources: types.ResourceSet{
					Name:      "test-resources",
					Resources: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "error when failed to apply resources",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
					return &clusters.ProviderMock{
						ApplyResourcesFunc: func(clusterSpec *types.ClusterSpec, resources types.ResourceSet) (*types.ResourceSet, error) {
							return nil, errors.Errorf("failed to apply resources")
						},
					}, nil
				}},
			},
			args: args{
				cluster: cluster,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			err := c.ApplyResources(tt.args.cluster, tt.args.resources)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyResources() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClusterService_InstallDinosaurOperator(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}
	type args struct {
		cluster *api.Cluster
		addonID string
	}

	clusterID := "test-internal-id"
	clusterExternalID := "test-external-id"
	clusterStatus := api.ClusterProvisioning

	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterID,
		},
		ExternalID:   clusterExternalID,
		ClusterID:    clusterID,
		Status:       clusterStatus,
		ProviderType: api.ClusterProviderOCM,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    bool
	}{
		{
			name: "successfully install dinosaur operator",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{InstallDinosaurOperatorFunc: func(clusterSpec *types.ClusterSpec) (bool, error) {
							return true, nil
						}}, nil
					},
				},
			},
			args: args{
				cluster: cluster,
				addonID: "test-id",
			},
			wantErr: false,
			want:    true,
		},
		{
			name: "error when failed to install dinosaur operator",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							InstallDinosaurOperatorFunc: func(clusterSpec *types.ClusterSpec) (bool, error) {
								return false, errors.Errorf("failed to install addon")
							}}, nil
					},
				},
			},
			args: args{
				cluster: cluster,
				addonID: "test-addon-id",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.InstallDinosaurOperator(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("InstallDinosaurOperator() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InstallDinosaurOperator want %v, but got %v", tt.want, got)
			}
		})
	}
}

func Test_ClusterService_GetExternalID(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		clusterProviderFactory clusters.ProviderFactory
	}

	type args struct {
		clusterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setupFn func()
		wantErr bool
		want    string
	}{
		{
			name: "When cluster exists and external ID exists it is returned",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			setupFn: func() {
				mockedResponse := []map[string]interface{}{{"external_id": "test-cluster-id"}}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithReply(mockedResponse)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: false,
			want:    "test-cluster-id",
		},
		{
			name: "When cluster exists and external ID does not exit the empty string is returned",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			setupFn: func() {
				mockedResponse := []map[string]interface{}{{"external_id": ""}}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithReply(mockedResponse)
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			wantErr: false,
			want:    "",
		},
		{
			name: "When cluster does not exist an error is returned",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM "clusters" WHERE "clusters"."cluster_id" = $1`).
					WithArgs(testClusterID).
					WithError(gorm.ErrRecordNotFound)
			},
			wantErr: true,
		},
		{
			name: "When provided clusterID is empty an error is returned",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				clusterProviderFactory: &clusters.ProviderFactoryMock{},
			},
			args: args{
				clusterID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			c := &clusterService{
				connectionFactory: tt.fields.connectionFactory,
				providerFactory:   tt.fields.clusterProviderFactory,
			}

			got, err := c.GetExternalID(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetExternalID() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetExternalID() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}
