package services

import (
	"encoding/json"
	"errors"

	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/clusters"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/clusters/types"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"

	"gorm.io/gorm"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	apiErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// ClusterService ...
//
//go:generate moq -out clusterservice_moq.go . ClusterService
type ClusterService interface {
	Create(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError)
	GetClusterDNS(clusterID string) (string, *apiErrors.ServiceError)
	GetExternalID(clusterID string) (string, *apiErrors.ServiceError)
	ListByStatus(state api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError)
	UpdateStatus(cluster api.Cluster, status api.ClusterStatus) error
	// Update updates a Cluster. Only fields whose value is different than the
	// zero-value of their corresponding type will be updated
	Update(cluster api.Cluster) *apiErrors.ServiceError
	// Updates() updates the given fields of a Clister. This takes in a map so that even zero-fields can be updated.
	// Use this only when you want to update the multiple columns that may contain zero-fields, otherwise use the `ClusterService.Update()` method.
	// See https://gorm.io/docs/update.html#Updates-multiple-columns for more info
	Updates(cluster api.Cluster, values map[string]interface{}) *apiErrors.ServiceError
	FindCluster(criteria FindClusterCriteria) (*api.Cluster, *apiErrors.ServiceError)
	// FindClusterByID returns the cluster corresponding to the provided clusterID.
	// If the cluster has not been found nil is returned. If there has been an issue
	// finding the cluster an error is set
	FindClusterByID(clusterID string) (*api.Cluster, *apiErrors.ServiceError)
	ListGroupByProviderAndRegion(providers []string, regions []string, status []string) ([]*ResGroupCPRegion, *apiErrors.ServiceError)
	RegisterClusterJob(clusterRequest *api.Cluster) *apiErrors.ServiceError
	// DeleteByClusterID will delete the cluster from the database
	DeleteByClusterID(clusterID string) *apiErrors.ServiceError
	// FindNonEmptyClusterByID returns a cluster if it present and it is not empty.
	// Cluster emptiness is determined by checking whether the cluster contains Centrals that have been provisioned, are being provisioned on it, or are being deprovisioned from it i.e central that are not in failure state.
	FindNonEmptyClusterByID(clusterID string) (*api.Cluster, *apiErrors.ServiceError)
	// ListAllClusterIds returns all the valid cluster ids in array
	ListAllClusterIds() ([]api.Cluster, *apiErrors.ServiceError)
	// FindAllClusters return all the valid clusters in array
	FindAllClusters(criteria FindClusterCriteria) ([]*api.Cluster, *apiErrors.ServiceError)
	// FindCentralInstanceCount returns the central instance counts associated with the list of clusters. If the list is empty, it will list all clusterIds that have Central instances assigned.
	FindCentralInstanceCount(clusterIDs []string) ([]ResCentralInstanceCount, *apiErrors.ServiceError)
	// UpdateMultiClusterStatus updates a list of clusters' status to a status
	UpdateMultiClusterStatus(clusterIds []string, status api.ClusterStatus) *apiErrors.ServiceError
	// CountByStatus returns the count of clusters for each given status in the database
	CountByStatus([]api.ClusterStatus) ([]ClusterStatusCount, *apiErrors.ServiceError)
	CheckClusterStatus(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError)
	// Delete will delete the cluster from the provider
	Delete(cluster *api.Cluster) (bool, *apiErrors.ServiceError)
}

type clusterService struct {
	connectionFactory *db.ConnectionFactory
	providerFactory   clusters.ProviderFactory
}

// NewClusterService creates a new client for the OSD Cluster Service
func NewClusterService(connectionFactory *db.ConnectionFactory, providerFactory clusters.ProviderFactory) ClusterService {
	return &clusterService{
		connectionFactory: connectionFactory,
		providerFactory:   providerFactory,
	}
}

// RegisterClusterJob registers a new job in the cluster table
func (c clusterService) RegisterClusterJob(clusterRequest *api.Cluster) *apiErrors.ServiceError {
	dbConn := c.connectionFactory.New()
	if err := dbConn.Save(clusterRequest).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to register cluster job")
	}
	return nil
}

// Create Creates a new OpenShift/k8s cluster via the provider and save the details of the cluster in the database
// Returns the newly created cluster object
func (c clusterService) Create(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()
	r := &types.ClusterRequest{
		CloudProvider:  cluster.CloudProvider,
		Region:         cluster.Region,
		MultiAZ:        cluster.MultiAZ,
		AdditionalSpec: cluster.ProviderSpec,
	}
	provider, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}
	clusterSpec, err := provider.Create(r)
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to create cluster")
	}

	cluster.ClusterID = clusterSpec.InternalID
	cluster.ExternalID = clusterSpec.ExternalID
	cluster.Status = clusterSpec.Status
	cluster.Schedulable = true
	if clusterSpec.AdditionalInfo != nil {
		clusterInfo, err := json.Marshal(clusterSpec.AdditionalInfo)
		if err != nil {
			return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to marshal JSON value")
		}
		cluster.ClusterSpec = clusterInfo
	}

	if err := dbConn.Save(cluster).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to save data to db")
	}

	return cluster, nil
}

// GetClusterDNS gets an OSD clusters DNS from OCM cluster service by ID
//
// Returns the DNS name
func (c clusterService) GetClusterDNS(clusterID string) (string, *apiErrors.ServiceError) {
	cluster, serviceErr := c.FindClusterByID(clusterID)
	if serviceErr != nil {
		return "", serviceErr
	}

	if cluster != nil && cluster.ClusterDNS != "" {
		return cluster.ClusterDNS, nil
	}

	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return "", apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}

	// If the clusterDNS is not present in the database, retrieve it from OCM
	clusterDNS, err := p.GetClusterDNS(buildClusterSpec(cluster))
	if err != nil {
		return "", apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get cluster DNS from OCM")
	}
	cluster.ClusterDNS = clusterDNS
	if err := c.Update(*cluster); err != nil {
		return "", apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update cluster DNS")
	}
	return clusterDNS, nil
}

// ListByStatus ...
func (c clusterService) ListByStatus(status api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
	if status.String() == "" {
		return nil, apiErrors.Validation("status is undefined")
	}
	dbConn := c.connectionFactory.New()

	var clusters []api.Cluster

	if err := dbConn.Model(&api.Cluster{}).Where("status = ?", status).Scan(&clusters).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to query by status")
	}

	return clusters, nil
}

// Update ...
func (c clusterService) Update(cluster api.Cluster) *apiErrors.ServiceError {
	if cluster.ID == "" {
		return apiErrors.Validation("id is undefined")
	}

	// by specifying the Model with a non-empty primary key we ensure
	// only the record with that primary key is updated
	dbConn := c.connectionFactory.New().Model(cluster)

	if err := dbConn.Updates(cluster).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update cluster")
	}

	return nil
}

// Updates ...
func (c clusterService) Updates(cluster api.Cluster, fields map[string]interface{}) *apiErrors.ServiceError {
	if cluster.ID == "" {
		return apiErrors.Validation("id is undefined")
	}

	// by specifying the Model with a non-empty primary key we ensure
	// only the record with that primary key is updated
	dbConn := c.connectionFactory.New().Model(cluster)

	if err := dbConn.Updates(fields).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update cluster")
	}

	return nil
}

// UpdateStatus ...
func (c clusterService) UpdateStatus(cluster api.Cluster, status api.ClusterStatus) error {
	if status.String() == "" {
		return apiErrors.Validation("status is undefined")
	}
	if cluster.ID == "" && cluster.ClusterID == "" {
		return apiErrors.Validation("id is undefined")
	}

	if status == api.ClusterReady || status == api.ClusterFailed {
		metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationCreate)
	}

	dbConn := c.connectionFactory.New()

	var query, arg string

	if cluster.ID != "" {
		query, arg = "id = ?", cluster.ID
	} else {
		query, arg = "cluster_id = ?", cluster.ClusterID
	}

	if err := dbConn.Model(&api.Cluster{}).Where(query, arg).Update("status", status).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update cluster status")
	}

	if status == api.ClusterReady {
		metrics.IncreaseClusterSuccessOperationsCountMetric(constants.ClusterOperationCreate)
	}

	return nil
}

// ResGroupCPRegion ...
type ResGroupCPRegion struct {
	Provider string
	Region   string
	Count    int
}

// ListGroupByProviderAndRegion retrieves existing OSD cluster with specified status in all providers and regions
func (c clusterService) ListGroupByProviderAndRegion(providers []string, regions []string, status []string) ([]*ResGroupCPRegion, *apiErrors.ServiceError) {
	if len(providers) == 0 || len(regions) == 0 || len(status) == 0 {
		return nil, apiErrors.Validation("provider, region and status must not be empty")
	}
	dbConn := c.connectionFactory.New()
	var grpResult []*ResGroupCPRegion

	// only one record returns for each region if they exist
	if err := dbConn.Model(&api.Cluster{}).
		Select("cloud_provider as Provider, region as Region, count(1) as Count").
		Where("cloud_provider in (?)", providers).
		Where("region in (?)", regions).
		Where("status in (?) ", status).
		Group("cloud_provider, region").Scan(&grpResult).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to list by cloud provider, regions and status")
	}

	return grpResult, nil
}

// FindClusterCriteria ...
type FindClusterCriteria struct {
	Provider string
	Region   string
	MultiAZ  bool
	Status   api.ClusterStatus
}

// FindCluster ...
func (c clusterService) FindCluster(criteria FindClusterCriteria) (*api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var cluster api.Cluster

	clusterDetails := &api.Cluster{
		CloudProvider: criteria.Provider,
		Region:        criteria.Region,
		MultiAZ:       criteria.MultiAZ,
		Status:        criteria.Status,
	}

	// we order them by "created_at" field instead of the default "id" field.
	// They are mostly the same as the library we use (xid) does take the generation timestamp into consideration,
	// However, it only down to the level of seconds. This means that if a few records are created at almost the same time,
	// the order is not guaranteed. So use the `created_at` column will provider better consistency.
	if err := dbConn.Where(clusterDetails).First(&cluster).Order("created_at asc").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to find cluster with criteria")
	}

	return &cluster, nil
}

// FindClusterByID ...
func (c clusterService) FindClusterByID(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
	if clusterID == "" {
		return nil, apiErrors.Validation("clusterID is undefined")
	}
	dbConn := c.connectionFactory.New()

	cluster := &api.Cluster{}

	clusterDetails := &api.Cluster{
		ClusterID: clusterID,
	}

	if err := dbConn.Where(clusterDetails).First(cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to find cluster with id: %s", clusterID)
	}

	return cluster, nil
}

// DeleteByClusterID ...
func (c clusterService) DeleteByClusterID(clusterID string) *apiErrors.ServiceError {
	dbConn := c.connectionFactory.New()
	metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationDelete)

	if err := dbConn.Delete(&api.Cluster{}, api.Cluster{ClusterID: clusterID}).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "Unable to delete cluster with cluster_id %s", clusterID)
	}

	glog.Infof("Cluster %s deleted successful", clusterID)
	metrics.IncreaseClusterSuccessOperationsCountMetric(constants.ClusterOperationDelete)
	return nil
}

// FindNonEmptyClusterByID ...
func (c clusterService) FindNonEmptyClusterByID(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	cluster := &api.Cluster{}

	clusterDetails := &api.Cluster{
		ClusterID: clusterID,
	}

	subQuery := dbConn.Select("cluster_id").Where("status != ? AND cluster_id = ?", constants.CentralRequestStatusFailed, clusterID).Model(dbapi.CentralRequest{})
	if err := dbConn.Where(clusterDetails).Where("cluster_id IN (?)", subQuery).First(cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to find cluster with id %s", clusterID)
	}

	return cluster, nil
}

// ListAllClusterIds ...
func (c clusterService) ListAllClusterIds() ([]api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var res []api.Cluster

	// we order them by "created_at" field instead of the default "id" field.
	// They are mostly the same as the library we use (xid) does take the generation timestamp into consideration,
	// However, it only down to the level of seconds. This means that if a few records are created at almost the same time,
	// the order is not guaranteed. So use the `created_at` column will provider better consistency.
	if err := dbConn.Model(&api.Cluster{}).
		Select("cluster_id").
		Where("cluster_id != '' ").
		Order("created_at asc ").
		Scan(&res).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to query by cluster info")
	}
	return res, nil
}

// ResCentralInstanceCount ...
type ResCentralInstanceCount struct {
	Clusterid string
	Count     int
}

// GetExternalID ...
func (c clusterService) GetExternalID(clusterID string) (string, *apiErrors.ServiceError) {
	cluster, err := c.FindClusterByID(clusterID)
	if err != nil {
		return "", err
	}
	if cluster == nil {
		return "", apiErrors.GeneralError("failed to get External ID for clusterID %s", clusterID)
	}
	return cluster.ExternalID, nil
}

// FindCentralInstanceCount ...
func (c clusterService) FindCentralInstanceCount(clusterIDs []string) ([]ResCentralInstanceCount, *apiErrors.ServiceError) {
	var res []ResCentralInstanceCount
	query := c.connectionFactory.New().
		Model(&dbapi.CentralRequest{}).
		Select("cluster_id as Clusterid, count(1) as Count").
		Where("status != ?", constants.CentralRequestStatusAccepted.String()) // central in accepted state do not have a cluster_id assigned to them

	if len(clusterIDs) > 0 {
		query = query.Where("cluster_id in (?)", clusterIDs)
	} else {
		query = query.Where("cluster_id != ''") // make sure that we only include central having a cluster_id
	}

	query = query.Group("cluster_id").Order("cluster_id asc").Scan(&res)

	if err := query.Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to query by cluster info")
	}
	// the query above won't return a count for a clusterId if that cluster doesn't have any Centrals,
	// to keep things consistent and less confusing, we will identity these ids and set their count to 0
	if len(clusterIDs) > 0 {
		countersMap := map[string]int{}
		for _, c := range res {
			countersMap[c.Clusterid] = c.Count
		}
		for _, clusterID := range clusterIDs {
			if _, ok := countersMap[clusterID]; !ok {
				res = append(res, ResCentralInstanceCount{Clusterid: clusterID, Count: 0})
			}
		}
	}

	return res, nil
}

// FindAllClusters ...
func (c clusterService) FindAllClusters(criteria FindClusterCriteria) ([]*api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New().
		Model(&api.Cluster{})

	var cluster []*api.Cluster

	clusterDetails := &api.Cluster{
		CloudProvider: criteria.Provider,
		Region:        criteria.Region,
		MultiAZ:       criteria.MultiAZ,
		Status:        criteria.Status,
	}

	// we order them by "created_at" field instead of the default "id" field.
	// They are mostly the same as the library we use (xid) does take the generation timestamp into consideration,
	// However, it only down to the level of seconds. This means that if a few records are created at almost the same time,
	// the order is not guaranteed. So use the `created_at` column will provider better consistency.
	if err := dbConn.Where(clusterDetails).Order("created_at asc").Scan(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to find all clusters with criteria")
	}

	return cluster, nil
}

// UpdateMultiClusterStatus ...
func (c clusterService) UpdateMultiClusterStatus(clusterIds []string, status api.ClusterStatus) *apiErrors.ServiceError {
	if status.String() == "" {
		return apiErrors.Validation("status is undefined")
	}
	if len(clusterIds) == 0 {
		return apiErrors.Validation("ids is empty")
	}

	dbConn := c.connectionFactory.New().
		Model(&api.Cluster{}).
		Where("cluster_id in (?)", clusterIds)

	if status == api.ClusterDeprovisioning {
		dbConn = dbConn.Where("status != ?", api.ClusterCleanup.String())
	}

	if err := dbConn.Update("status", status).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update status: %s", clusterIds)
	}

	for rows := dbConn.RowsAffected; rows > 0; rows-- {
		if status == api.ClusterFailed {
			metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationCreate)
		}
		if status == api.ClusterReady {
			metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationCreate)
			metrics.IncreaseClusterSuccessOperationsCountMetric(constants.ClusterOperationCreate)
		}
	}

	return nil
}

// ClusterStatusCount ...
type ClusterStatusCount struct {
	Status api.ClusterStatus
	Count  int
}

// CountByStatus ...
func (c clusterService) CountByStatus(status []api.ClusterStatus) ([]ClusterStatusCount, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()
	var results []ClusterStatusCount
	if err := dbConn.Model(&api.Cluster{}).Select("status as Status, count(1) as Count").Where("status in (?)", status).Group("status").Scan(&results).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to count by status")
	}

	// if there is no count returned for a status from the above query because there is no clusters in such a status,
	// we should return the count for these as well to avoid any confusion
	if len(status) > 0 {
		countersMap := map[api.ClusterStatus]int{}
		for _, c := range results {
			countersMap[c.Status] = c.Count
		}
		for _, s := range status {
			if _, ok := countersMap[s]; !ok {
				results = append(results, ClusterStatusCount{Status: s, Count: 0})
			}
		}
	}

	return results, nil
}

// CheckClusterStatus ...
func (c clusterService) CheckClusterStatus(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}

	clusterSpec, err := p.CheckClusterStatus(buildClusterSpec(cluster))
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to check cluster status")
	}

	cluster.Status = clusterSpec.Status
	cluster.StatusDetails = clusterSpec.StatusDetails
	cluster.ClusterSpec = clusterSpec.AdditionalInfo
	if clusterSpec.ExternalID != "" && cluster.ExternalID == "" {
		cluster.ExternalID = clusterSpec.ExternalID
	}
	if err := c.Update(*cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// Delete ...
func (c clusterService) Delete(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return false, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}
	removed, err := p.Delete(buildClusterSpec(cluster))
	if err != nil {
		return false, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to delete the cluster from the provider")
	}
	return removed, nil
}

func buildClusterSpec(cluster *api.Cluster) *types.ClusterSpec {
	return &types.ClusterSpec{
		InternalID:     cluster.ClusterID,
		ExternalID:     cluster.ExternalID,
		Status:         cluster.Status,
		AdditionalInfo: cluster.ClusterSpec,
	}
}
