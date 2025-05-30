// Package workers ...
package workers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/gitops"
	ocm "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"

	"github.com/goava/di"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"

	"github.com/pkg/errors"
)

const (
	mkReadOnlyGroupName             = "mk-readonly-access"
	mkSREGroupName                  = "central-sre"
	mkReadOnlyRoleBindingName       = "mk-dedicated-readers"
	mkSRERoleBindingName            = "central-sre-cluster-admin"
	dedicatedReadersRoleBindingName = "dedicated-readers"
	clusterAdminRoleName            = "cluster-admin"
)

var (
	readyClusterCount int32
)

var clusterMetricsStatuses = []api.ClusterStatus{
	api.ClusterAccepted,
	api.ClusterProvisioning,
	api.ClusterProvisioned,
	api.ClusterCleanup,
	api.ClusterReady,
	api.ClusterComputeNodeScalingUp,
	api.ClusterFull,
	api.ClusterFailed,
	api.ClusterDeprovisioning,
}

// Worker ...
type Worker = workers.Worker

// ClusterManager represents a cluster manager that periodically reconciles osd clusters

// ClusterManager ...
type ClusterManager struct {
	id           string
	workerType   string
	isRunning    bool
	imStop       chan struct{} // a chan used only for cancellation
	syncTeardown sync.WaitGroup
	ClusterManagerOptions
}

// ClusterManagerOptions ...
type ClusterManagerOptions struct {
	di.Inject
	Reconciler             workers.Reconciler
	OCMConfig              *ocm.OCMConfig
	DataplaneClusterConfig *config.DataplaneClusterConfig
	SupportedProviders     *config.ProviderConfig
	ClusterService         services.ClusterService
	CloudProvidersService  services.CloudProvidersService
	AddonProvisioner       *services.AddonProvisioner
	GitOpsConfigProvider   gitops.ConfigProvider
}

type processor func() []error

// NewClusterManager creates a new cluster manager
func NewClusterManager(o ClusterManagerOptions) *ClusterManager {
	return &ClusterManager{
		id:                    uuid.New().String(),
		workerType:            "cluster",
		ClusterManagerOptions: o,
	}
}

// GetStopChan ...
func (c *ClusterManager) GetStopChan() *chan struct{} {
	return &c.imStop
}

// GetSyncGroup ...
func (c *ClusterManager) GetSyncGroup() *sync.WaitGroup {
	return &c.syncTeardown
}

// GetID returns the ID that represents this worker
func (c *ClusterManager) GetID() string {
	return c.id
}

// GetWorkerType ...
func (c *ClusterManager) GetWorkerType() string {
	return c.workerType
}

func (c *ClusterManager) GetRepeatInterval() time.Duration {
	return workers.DefaultRepeatInterval
}

// Start initializes the cluster manager to reconcile osd clusters
func (c *ClusterManager) Start() {
	metrics.SetLeaderWorkerMetric(c.workerType, true)
	c.Reconciler.Start(c)
}

// Stop causes the process for reconciling osd clusters to stop.
func (c *ClusterManager) Stop() {
	glog.Infof("Stopping reconciling cluster manager id = %s", c.id)
	c.Reconciler.Stop(c)
	metrics.ResetMetricsForClusterManagers()
	metrics.SetLeaderWorkerMetric(c.workerType, false)
}

// IsRunning ...
func (c *ClusterManager) IsRunning() bool {
	return c.isRunning
}

// SetIsRunning ...
func (c *ClusterManager) SetIsRunning(val bool) {
	c.isRunning = val
}

// Reconcile ...
func (c *ClusterManager) Reconcile() []error {
	var encounteredErrors []error

	processors := []processor{
		c.processMetrics,
		c.reconcileClusterWithManualConfig,
		c.reconcileClustersForRegions,
		c.processDeprovisioningClusters,
		c.processCleanupClusters,
		c.processAcceptedClusters,
		c.processProvisioningClusters,
		c.processProvisionedClusters,
		c.processReadyClusters,
	}

	for _, p := range processors {
		if errs := p(); len(errs) > 0 {
			encounteredErrors = append(encounteredErrors, errs...)
		}
	}
	return encounteredErrors
}

func (c *ClusterManager) processMetrics() []error {
	if err := c.setClusterStatusCountMetrics(); err != nil {
		return []error{errors.Wrapf(err, "failed to set cluster status count metrics")}
	}

	if err := c.setCentralPerClusterCountMetrics(); err != nil {
		return []error{errors.Wrapf(err, "failed to set central per cluster count metrics")}
	}

	c.setClusterStatusMaxCapacityMetrics()

	return []error{}
}

func (c *ClusterManager) processDeprovisioningClusters() []error {
	var errs []error
	deprovisioningClusters, serviceErr := c.ClusterService.ListByStatus(api.ClusterDeprovisioning)
	if serviceErr != nil {
		errs = append(errs, serviceErr)
		return errs
	}
	if len(deprovisioningClusters) > 0 {
		glog.Infof("deprovisioning clusters count = %d", len(deprovisioningClusters))
	}

	for i := range deprovisioningClusters {
		cluster := deprovisioningClusters[i]
		glog.V(10).Infof("deprovision cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterDeprovisioning)
		if err := c.reconcileDeprovisioningCluster(&cluster); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile deprovisioning cluster %s", cluster.ID))
		}
	}
	return errs
}

func (c *ClusterManager) processCleanupClusters() []error {
	var errs []error
	cleanupClusters, serviceErr := c.ClusterService.ListByStatus(api.ClusterCleanup)
	if serviceErr != nil {
		errs = append(errs, errors.Wrap(serviceErr, "failed to list of cleaup clusters"))
		return errs
	}

	if len(cleanupClusters) > 0 {
		glog.Infof("cleanup clusters count = %d", len(cleanupClusters))
	}

	for _, cluster := range cleanupClusters {
		glog.V(10).Infof("cleanup cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterCleanup)
		if err := c.reconcileCleanupCluster(cluster); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile cleanup cluster %s", cluster.ID))
		}
	}
	return errs
}

func (c *ClusterManager) processAcceptedClusters() []error {
	var errs []error
	acceptedClusters, serviceErr := c.ClusterService.ListByStatus(api.ClusterAccepted)
	if serviceErr != nil {
		errs = append(errs, errors.Wrap(serviceErr, "failed to list accepted clusters"))
		return errs
	}

	if len(acceptedClusters) > 0 {
		glog.Infof("accepted clusters count = %d", len(acceptedClusters))
	}

	for i := range acceptedClusters {
		cluster := acceptedClusters[i]
		glog.V(10).Infof("accepted cluster ClusterID = %s", cluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(cluster, api.ClusterAccepted)
		if err := c.reconcileAcceptedCluster(&cluster); err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile accepted cluster %s", cluster.ID))
			continue
		}
	}
	return errs
}

func (c *ClusterManager) processProvisioningClusters() []error {
	var errs []error
	provisioningClusters, listErr := c.ClusterService.ListByStatus(api.ClusterProvisioning)
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list pending clusters"))
		return errs
	}
	if len(provisioningClusters) > 0 {
		glog.Infof("provisioning clusters count = %d", len(provisioningClusters))
	}

	// process each local pending cluster and compare to the underlying ocm cluster
	for i := range provisioningClusters {
		provisioningCluster := provisioningClusters[i]
		glog.V(10).Infof("provisioning cluster ClusterID = %s", provisioningCluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(provisioningCluster, api.ClusterProvisioning)
		_, err := c.reconcileClusterStatus(&provisioningCluster)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile cluster %s status", provisioningCluster.ClusterID))
			continue
		}
	}
	return errs
}

func (c *ClusterManager) processProvisionedClusters() []error {
	var errs []error
	/*
	 * Terraforming Provisioned Clusters
	 */
	provisionedClusters, listErr := c.ClusterService.ListByStatus(api.ClusterProvisioned)
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list provisioned clusters"))
		return errs
	}
	if len(provisionedClusters) > 0 {
		glog.Infof("provisioned clusters count = %d", len(provisionedClusters))
	}

	// process each local provisioned cluster and apply necessary terraforming
	for _, provisionedCluster := range provisionedClusters {
		glog.V(10).Infof("provisioned cluster ClusterID = %s", provisionedCluster.ClusterID)
		metrics.UpdateClusterStatusSinceCreatedMetric(provisionedCluster, api.ClusterProvisioned)
		err := c.reconcileProvisionedCluster(provisionedCluster)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed to reconcile provisioned cluster %s", provisionedCluster.ClusterID))
			continue
		}
	}

	return errs
}

func (c *ClusterManager) processReadyClusters() []error {
	var errs []error
	// Keep SyncSet up to date for clusters that are ready
	readyClusters, listErr := c.ClusterService.ListByStatus(api.ClusterReady)
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list ready clusters"))
		return errs
	}

	readyClusterCount = int32(len(readyClusters))
	logger.InfoChangedInt32(&readyClusterCount, "ready clusters count = %d", readyClusterCount)

	gitopsConfig, err := c.GitOpsConfigProvider.Get()
	if err != nil {
		errs = append(errs, fmt.Errorf("get gitops config: %w", err))
		return errs
	}
	clusterConfigByID := make(map[string]gitops.DataPlaneClusterConfig)
	for _, cluster := range gitopsConfig.DataPlaneClusters {
		clusterConfigByID[cluster.ClusterID] = cluster
	}

	for _, readyCluster := range readyClusters {
		emptyClusterReconciled := false
		var recErr error
		if c.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
			emptyClusterReconciled, recErr = c.reconcileEmptyCluster(readyCluster)
		}
		if !emptyClusterReconciled && recErr == nil {
			recErr = c.reconcileReadyCluster(readyCluster)
		}
		if recErr == nil {
			recErr = c.reconcileClusterAddons(readyCluster, clusterConfigByID)
		}
		if recErr != nil {
			errs = append(errs, errors.Wrapf(recErr, "failed to reconcile ready cluster %s", readyCluster.ClusterID))
		}
	}
	return errs
}

func (c *ClusterManager) reconcileDeprovisioningCluster(cluster *api.Cluster) error {
	if c.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		siblingCluster, findClusterErr := c.ClusterService.FindCluster(services.FindClusterCriteria{
			Region:   cluster.Region,
			Provider: cluster.CloudProvider,
			MultiAZ:  cluster.MultiAZ,
			Status:   api.ClusterReady,
		})

		if findClusterErr != nil {
			return findClusterErr
		}

		// if it is the only cluster left in that region, set it back to ready.
		if siblingCluster == nil {
			err := c.ClusterService.UpdateStatus(*cluster, api.ClusterReady)
			if err != nil {
				return fmt.Errorf("updating status for cluster %s to %s: %w", cluster.ClusterID, api.ClusterReady, err)
			}
			return nil
		}
	}

	deleted, deleteClusterErr := c.ClusterService.Delete(cluster)
	if deleteClusterErr != nil {
		return deleteClusterErr
	}

	if !deleted {
		return nil
	}

	// cluster has been removed from cluster service. Mark it for cleanup
	glog.Infof("Cluster %s  has been removed from cluster service.", cluster.ClusterID)
	updateStatusErr := c.ClusterService.UpdateStatus(*cluster, api.ClusterCleanup)
	if updateStatusErr != nil {
		return errors.Wrapf(updateStatusErr, "Failed to update deprovisioning cluster %s status to 'cleanup'", cluster.ClusterID)
	}

	return nil
}

func (c *ClusterManager) reconcileCleanupCluster(cluster api.Cluster) error {
	glog.Infof("Removing Dataplane cluster %s fleetshard service account", cluster.ClusterID)

	glog.Infof("Soft deleting the Dataplane cluster %s from the database", cluster.ClusterID)
	deleteError := c.ClusterService.DeleteByClusterID(cluster.ClusterID)
	if deleteError != nil {
		return errors.Wrapf(deleteError, "Failed to soft delete Dataplane cluster %s from the database", cluster.ClusterID)
	}
	return nil
}

func (c *ClusterManager) reconcileReadyCluster(cluster api.Cluster) error {
	if !c.DataplaneClusterConfig.IsReadyDataPlaneClustersReconcileEnabled() {
		glog.Infof("Reconcile of dataplane ready clusters is disabled. Skipped reconcile of ready ClusterID '%s'", cluster.ClusterID)
		return nil
	}

	var err error

	err = c.reconcileClusterInstanceType(cluster)
	if err != nil {
		return errors.WithMessagef(err, "failed to reconcile instance type ready cluster %s: %s", cluster.ClusterID, err.Error())
	}

	// TODO: Register what is necessary for SSO authn/authz.
	// err = c.reconcileClusterIdentityProvider(cluster)
	// if err != nil {
	//	return errors.WithMessagef(err, "failed to reconcile identity provider of ready cluster %s: %s", cluster.ClusterID, err.Error())
	//}

	err = c.reconcileClusterDNS(cluster)
	if err != nil {
		return errors.WithMessagef(err, "failed to reconcile cluster dns of ready cluster %s: %s", cluster.ClusterID, err.Error())
	}

	return nil
}

// reconcileClusterInstanceType checks whether a cluster has an instance type, if not, set to the instance type provided in the manual cluster configuration
// If the cluster does not exist, assume the cluster supports both instance types
func (c *ClusterManager) reconcileClusterInstanceType(cluster api.Cluster) error {
	supportedInstanceType := api.AllInstanceTypeSupport.String()
	manualScalingEnabled := c.DataplaneClusterConfig.IsDataPlaneManualScalingEnabled()
	if manualScalingEnabled {
		supportedType, found := c.DataplaneClusterConfig.ClusterConfig.GetClusterSupportedInstanceType(cluster.ClusterID)
		if !found && cluster.SupportedInstanceType != "" {
			logger.Logger.Infof("cluster instance type already set for cluster = %s", cluster.ClusterID)
			return nil
		} else if found {
			supportedInstanceType = supportedType
		}
	}

	if cluster.SupportedInstanceType != "" && !manualScalingEnabled {
		logger.Logger.Infof("cluster instance type already set for cluster = %s and scaling type is not manual", cluster.ClusterID)
		return nil
	}

	if cluster.SupportedInstanceType != supportedInstanceType {
		cluster.SupportedInstanceType = supportedInstanceType
		err := c.ClusterService.Update(cluster)
		if err != nil {
			return errors.Wrapf(err, "failed to update instance type in database for cluster %s", cluster.ClusterID)
		}
		logger.Logger.V(10).Infof("supported instance type for cluster = %s successful updated", cluster.ClusterID)
	}

	return nil
}

// reconcileEmptyCluster checks wether a cluster is empty and mark it for deletion
func (c *ClusterManager) reconcileEmptyCluster(cluster api.Cluster) (bool, error) {
	glog.V(10).Infof("check if cluster is empty, ClusterID = %s", cluster.ClusterID)
	clusterFromDb, err := c.ClusterService.FindNonEmptyClusterByID(cluster.ClusterID)
	if err != nil {
		return false, err
	}
	if clusterFromDb != nil {
		glog.V(10).Infof("cluster is not empty, ClusterID = %s", cluster.ClusterID)
		return false, nil
	}

	clustersByRegionAndCloudProvider, findSiblingClusterErr := c.ClusterService.ListGroupByProviderAndRegion(
		[]string{cluster.CloudProvider},
		[]string{cluster.Region},
		[]string{api.ClusterReady.String()})

	if findSiblingClusterErr != nil || len(clustersByRegionAndCloudProvider) == 0 {
		return false, findSiblingClusterErr
	}

	siblingClusterCount := clustersByRegionAndCloudProvider[0]
	if siblingClusterCount.Count <= 1 { // sibling cluster not found
		glog.V(10).Infof("no valid sibling found for cluster ClusterID = %s", cluster.ClusterID)
		return false, nil
	}

	updateStatusErr := c.ClusterService.UpdateStatus(cluster, api.ClusterDeprovisioning)
	if updateStatusErr != nil {
		return false, fmt.Errorf("updating status for cluster %s to %s: %w", cluster.ClusterID, api.ClusterDeprovisioning, updateStatusErr)
	}
	return true, nil
}

func (c *ClusterManager) reconcileProvisionedCluster(cluster api.Cluster) error {
	// TODO: Register what is necessary for SSO authn/authz.
	// if err := c.reconcileClusterIdentityProvider(cluster); err != nil {
	//	return err
	//}

	return c.reconcileClusterDNS(cluster)
}

func (c *ClusterManager) reconcileClusterDNS(cluster api.Cluster) error {
	// Return if the clusterDNS is already set
	if cluster.ClusterDNS != "" {
		return nil
	}

	_, dnsErr := c.ClusterService.GetClusterDNS(cluster.ClusterID)
	if dnsErr != nil {
		return errors.WithMessagef(dnsErr, "failed to reconcile cluster %s: GetClusterDNS %s", cluster.ClusterID, dnsErr.Error())
	}

	return nil
}

func (c *ClusterManager) reconcileAcceptedCluster(cluster *api.Cluster) error {
	_, err := c.ClusterService.Create(cluster)
	if err != nil {
		return errors.Wrapf(err, "failed to create cluster for request %s", cluster.ID)
	}

	return nil
}

// reconcileClusterStatus updates the provided clusters stored status to reflect it's current state
func (c *ClusterManager) reconcileClusterStatus(cluster *api.Cluster) (*api.Cluster, error) {
	updatedCluster, err := c.ClusterService.CheckClusterStatus(cluster)
	if err != nil {
		return nil, err
	}
	if updatedCluster.Status == api.ClusterFailed {
		metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, api.ClusterFailed)
		metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationCreate)
	}
	return updatedCluster, nil
}

// reconcileClusterWithConfig reconciles clusters within the dataplane-cluster-configuration file.
// New clusters will be registered if it is not yet in the database.
// A cluster will be deprovisioned if it is in the database but not in the coreConfig file.
func (c *ClusterManager) reconcileClusterWithManualConfig() []error {
	if !c.DataplaneClusterConfig.IsDataPlaneManualScalingEnabled() {
		glog.Infoln("manual cluster configuration reconciliation is skipped as it is disabled")
		return []error{}
	}

	allClusterIds, err := c.ClusterService.ListAllClusterIds()
	if err != nil {
		return []error{errors.Wrapf(err, "failed to retrieve cluster ids from clusters")}
	}
	clusterIdsMap := make(map[string]api.Cluster)
	for _, v := range allClusterIds {
		clusterIdsMap[v.ClusterID] = v
	}

	// Create all missing clusters
	for _, p := range c.DataplaneClusterConfig.ClusterConfig.MissingClusters(clusterIdsMap) {
		clusterRequest := api.Cluster{
			CloudProvider:         p.CloudProvider,
			Region:                p.Region,
			MultiAZ:               p.MultiAZ,
			ClusterID:             p.ClusterID,
			Status:                p.Status,
			ProviderType:          p.ProviderType,
			ClusterDNS:            p.ClusterDNS,
			SupportedInstanceType: p.SupportedInstanceType,
			Schedulable:           p.Schedulable,
		}

		if err := c.ClusterService.RegisterClusterJob(&clusterRequest); err != nil {
			return []error{errors.Wrapf(err, "Failed to register new cluster %s with config file", p.ClusterID)}
		}
		glog.Infof("Registered a new cluster with config file: %s ", p.ClusterID)
	}

	// Update existing clusters.
	for _, manualCluster := range c.DataplaneClusterConfig.ClusterConfig.ExistingClusters(clusterIdsMap) {
		cluster, err := c.ClusterService.FindClusterByID(manualCluster.ClusterID)
		if err != nil {
			glog.Warningf("Failed to lookup cluster %s in cluster service: %v", manualCluster.ClusterID, err)
			continue
		}

		newCluster := *cluster
		newCluster.CloudProvider = manualCluster.CloudProvider
		newCluster.Region = manualCluster.Region
		newCluster.MultiAZ = manualCluster.MultiAZ
		newCluster.Status = manualCluster.Status
		newCluster.ProviderType = manualCluster.ProviderType
		newCluster.ClusterDNS = manualCluster.ClusterDNS
		newCluster.SupportedInstanceType = manualCluster.SupportedInstanceType
		newCluster.Schedulable = manualCluster.Schedulable

		if cmp.Equal(*cluster, newCluster) {
			continue
		}
		diff := cmp.Diff(*cluster, newCluster)
		glog.Infof("Updating data-plane cluster %s. Changes in cluster configuration:\n", manualCluster.ClusterID)
		for _, diffLine := range strings.Split(diff, "\n") {
			glog.Infoln(diffLine)
		}

		// Gorm will not update primitive values if their new value is the same as their default value.
		// We therefore
		values := map[string]interface{}{
			"cloud_provider":          newCluster.CloudProvider,
			"region":                  newCluster.Region,
			"multi_az":                newCluster.MultiAZ,
			"status":                  newCluster.Status,
			"provider_type":           newCluster.ProviderType,
			"cluster_dns":             newCluster.ClusterDNS,
			"supported_instance_type": newCluster.SupportedInstanceType,
			"schedulable":             newCluster.Schedulable,
		}

		if err := c.ClusterService.Updates(newCluster, values); err != nil {
			return []error{errors.Wrapf(err, "Failed to update manual cluster %s", cluster.ClusterID)}
		}
	}

	// Remove all clusters that are not in the config file
	excessClusterIds := c.DataplaneClusterConfig.ClusterConfig.ExcessClusters(clusterIdsMap)
	if len(excessClusterIds) == 0 {
		return nil
	}

	centralInstanceCount, err := c.ClusterService.FindCentralInstanceCount(excessClusterIds)
	if err != nil {
		return []error{errors.Wrapf(err, "Failed to find central count a cluster: %s", excessClusterIds)}
	}

	var idsOfClustersToDeprovision []string
	for _, c := range centralInstanceCount {
		if c.Count > 0 {
			glog.Infof("Excess cluster %s is not going to be deleted because it has %d centrals.", c.Clusterid, c.Count)
		} else {
			glog.Infof("Excess cluster is going to be deleted %s", c.Clusterid)
			idsOfClustersToDeprovision = append(idsOfClustersToDeprovision, c.Clusterid)
		}
	}

	if len(idsOfClustersToDeprovision) == 0 {
		return nil
	}

	err = c.ClusterService.UpdateMultiClusterStatus(idsOfClustersToDeprovision, api.ClusterDeprovisioning)
	if err != nil {
		return []error{errors.Wrapf(err, "Failed to deprovisioning a cluster: %s", idsOfClustersToDeprovision)}
	}
	glog.Infof("Deprovisioning clusters: not found in config file: %s ", idsOfClustersToDeprovision)

	return []error{}
}

// reconcileClustersForRegions creates an OSD cluster for each supported cloud provider and region where no cluster exists.
func (c *ClusterManager) reconcileClustersForRegions() []error {
	var errs []error
	if !c.DataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
		return errs
	}
	glog.Infoln("reconcile cloud providers and regions")
	var providers []string
	var regions []string
	status := api.StatusForValidCluster
	// gather the supported providers and regions
	providerList := c.SupportedProviders.ProvidersConfig.SupportedProviders
	for _, v := range providerList {
		providers = append(providers, v.Name)
		for _, r := range v.Regions {
			regions = append(regions, r.Name)
		}
	}

	// get a list of clusters in Map group by their provider and region
	grpResult, err := c.ClusterService.ListGroupByProviderAndRegion(providers, regions, status)
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "failed to find cluster with criteria"))
		return errs
	}

	grpResultMap := make(map[string]*services.ResGroupCPRegion)
	for _, v := range grpResult {
		grpResultMap[v.Provider+"."+v.Region] = v
	}

	// create all the missing clusters in the supported provider and regions.
	for _, p := range providerList {
		for _, v := range p.Regions {
			if _, exist := grpResultMap[p.Name+"."+v.Name]; !exist {
				clusterRequest := api.Cluster{
					CloudProvider:         p.Name,
					Region:                v.Name,
					MultiAZ:               true,
					Status:                api.ClusterAccepted,
					ProviderType:          api.ClusterProviderOCM,
					SupportedInstanceType: api.AllInstanceTypeSupport.String(), // TODO - make sure we use the appropriate instance type.
				}
				if err := c.ClusterService.RegisterClusterJob(&clusterRequest); err != nil {
					errs = append(errs, errors.Wrapf(err, "Failed to auto-create cluster request in %s, region: %s", p.Name, v.Name))
					return errs
				}
				glog.Infof("Auto-created cluster request in %s, region: %s, Id: %s ", p.Name, v.Name, clusterRequest.ID)
			} //
		} // region
	} // provider
	return errs
}

func (c *ClusterManager) setClusterStatusMaxCapacityMetrics() {
	for _, cluster := range c.DataplaneClusterConfig.ClusterConfig.GetManualClusters() {
		supportedInstanceTypes := strings.Split(cluster.SupportedInstanceType, ",")
		for _, instanceType := range supportedInstanceTypes {
			if instanceType != "" {
				capacity := float64(cluster.CentralInstanceLimit)
				metrics.UpdateClusterStatusCapacityMaxCount(cluster.Region, instanceType, cluster.ClusterID, capacity)
			}
		}
	}
}

func (c *ClusterManager) setClusterStatusCountMetrics() error {
	counters, err := c.ClusterService.CountByStatus(clusterMetricsStatuses)
	if err != nil {
		return err
	}
	for _, c := range counters {
		metrics.UpdateClusterStatusCountMetric(c.Status, c.Count)
	}
	return nil
}

func (c *ClusterManager) setCentralPerClusterCountMetrics() error {
	counters, err := c.ClusterService.FindCentralInstanceCount([]string{})
	if err != nil {
		return err
	}
	for _, counter := range counters {
		clusterExternalID, err := c.ClusterService.GetExternalID(counter.Clusterid)
		if err != nil {
			return err
		}
		metrics.UpdateCentralPerClusterCountMetric(counter.Clusterid, clusterExternalID, counter.Count)
	}

	return nil
}

func (c *ClusterManager) reconcileClusterAddons(cluster api.Cluster, clusterConfigByID map[string]gitops.DataPlaneClusterConfig) error {
	clusterConfig, exists := clusterConfigByID[cluster.ClusterID]
	if !exists {
		// There's no such cluster in gitops config, skipping
		return nil
	}
	if err := c.AddonProvisioner.Provision(cluster, clusterConfig); err != nil {
		return fmt.Errorf("provision addons: %w", err)
	}
	return nil
}
