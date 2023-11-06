package presenters

import (
	"context"
	"fmt"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"sort"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/yaml"
)

// ManagedCentralPresenter helper service which converts Central DB representation to the private API representation
type ManagedCentralPresenter struct {
	centralConfig *config.CentralConfig
	gitopsConfig  gitops.ConfigProvider
	renderer      *cachedCentralRenderer
}

// NewManagedCentralPresenter creates a new instance of ManagedCentralPresenter
func NewManagedCentralPresenter(
	config *config.CentralConfig,
	gitopsConfig gitops.ConfigProvider,
) *ManagedCentralPresenter {
	return &ManagedCentralPresenter{
		centralConfig: config,
		gitopsConfig:  gitopsConfig,
		renderer:      newCachedCentralRenderer(),
	}
}

// PresentManagedCentrals converts DB representation of multiple Centrals to the private API representation in parallel
func (c *ManagedCentralPresenter) PresentManagedCentrals(ctx context.Context, from []*dbapi.CentralRequest) ([]private.ManagedCentral, error) {
	gitopsConfig, err := c.gitopsConfig.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get GitOps configuration")
	}
	ret := make([]private.ManagedCentral, len(from))
	g, ctx := errgroup.WithContext(ctx)
	const maxParallel = 50
	locks := make(chan struct{}, maxParallel)
	for i := range from {
		index := i
		g.Go(func() error {
			select {
			case locks <- struct{}{}:
			case <-ctx.Done():
				//nolint:wrapcheck
				return ctx.Err()
			}
			var err error
			ret[index], err = c.presentManagedCentral(gitopsConfig, from[index])
			<-locks

			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, errors.Wrap(err, "failed to convert central requests to managed centrals")
	}
	return ret, nil
}

// PresentManagedCentral converts DB representation of Central to the private API representation
func (c *ManagedCentralPresenter) PresentManagedCentral(from *dbapi.CentralRequest) (private.ManagedCentral, error) {
	gitopsConfig, err := c.gitopsConfig.Get()
	if err != nil {
		return private.ManagedCentral{}, errors.Wrap(err, "failed to get GitOps configuration")
	}
	return c.presentManagedCentral(gitopsConfig, from)
}

// PresentManagedCentralWithSecrets return a private.ManagedCentral including secret data
func (c *ManagedCentralPresenter) PresentManagedCentralWithSecrets(from *dbapi.CentralRequest) (private.ManagedCentral, error) {
	managedCentral, err := c.PresentManagedCentral(from)
	if err != nil {
		return private.ManagedCentral{}, err
	}
	secretInterfaceMap, err := from.Secrets.Object()
	secretStringMap := make(map[string]string, len(secretInterfaceMap))

	if err != nil {
		return managedCentral, errors.Wrapf(err, "failed to get Secrets for central request as map %q/%s", from.Name, from.ID)
	}

	for k, v := range secretInterfaceMap {
		secretStringMap[k] = fmt.Sprintf("%v", v)
	}

	managedCentral.Metadata.Secrets = secretStringMap // pragma: allowlist secret
	return managedCentral, nil
}

func (c *ManagedCentralPresenter) presentManagedCentral(gitopsConfig gitops.Config, from *dbapi.CentralRequest) (private.ManagedCentral, error) {
	centralParams := centralParamsFromRequest(from)
	centralYaml, err := c.renderer.getCentralYaml(gitopsConfig, centralParams)
	if err != nil {
		return private.ManagedCentral{}, errors.Wrap(err, "failed to get Central YAML")
	}

	res := private.ManagedCentral{
		Id:   from.ID,
		Kind: "ManagedCentral",
		Metadata: private.ManagedCentralAllOfMetadata{
			Name:      from.Name,
			Namespace: from.Namespace,
			Annotations: private.ManagedCentralAllOfMetadataAnnotations{
				MasId:          from.ID,
				MasPlacementId: from.PlacementID,
			},
			Internal:      from.Internal,
			SecretsStored: getSecretNames(from), // pragma: allowlist secret
		},
		Spec: private.ManagedCentralAllOfSpec{
			Owners: []string{
				from.Owner,
			},
			Auth: private.ManagedCentralAllOfSpecAuth{
				ClientId:     from.AuthConfig.ClientID,
				ClientSecret: from.AuthConfig.ClientSecret, // pragma: allowlist secret
				ClientOrigin: from.AuthConfig.ClientOrigin,
				OwnerOrgId:   from.OrganisationID,
				OwnerOrgName: from.OrganisationName,
				OwnerUserId:  from.OwnerUserID,
				Issuer:       from.AuthConfig.Issuer,
			},
			UiEndpoint: private.ManagedCentralAllOfSpecUiEndpoint{
				Host: from.GetUIHost(),
				Tls: private.ManagedCentralAllOfSpecUiEndpointTls{
					Cert: c.centralConfig.CentralTLSCert,
					Key:  c.centralConfig.CentralTLSKey,
				},
			},
			DataEndpoint: private.ManagedCentralAllOfSpecDataEndpoint{
				Host: from.GetDataHost(),
			},
			CentralCRYAML: string(centralYaml),
			InstanceType:  from.InstanceType,
		},
		RequestStatus: from.Status,
	}

	if from.DeletionTimestamp != nil {
		res.Metadata.DeletionTimestamp = from.DeletionTimestamp.Format(time.RFC3339)
	}

	return res, nil
}

func getSecretNames(from *dbapi.CentralRequest) []string {
	secrets, err := from.Secrets.Object()
	if err != nil {
		glog.Errorf("Failed to get Secrets as JSON object for Central request %q/%s: %v", from.Name, from.ClusterID, err)
		return []string{}
	}

	secretNames := make([]string, len(secrets))
	i := 0
	for k := range secrets {
		secretNames[i] = k
		i++
	}

	sort.Strings(secretNames)

	return secretNames
}

func centralParamsFromRequest(centralRequest *dbapi.CentralRequest) gitops.CentralParams {
	return gitops.CentralParams{
		ID:               centralRequest.ID,
		Name:             centralRequest.Name,
		Namespace:        centralRequest.Namespace,
		Region:           centralRequest.Region,
		ClusterID:        centralRequest.ClusterID,
		CloudProvider:    centralRequest.CloudProvider,
		CloudAccountID:   centralRequest.CloudAccountID,
		SubscriptionID:   centralRequest.SubscriptionID,
		Owner:            centralRequest.Owner,
		OwnerAccountID:   centralRequest.OwnerAccountID,
		OwnerUserID:      centralRequest.OwnerUserID,
		Host:             centralRequest.Host,
		OrganizationID:   centralRequest.OrganisationID,
		OrganizationName: centralRequest.OrganisationName,
		InstanceType:     centralRequest.InstanceType,
		IsInternal:       centralRequest.Internal,
	}
}

type renderFn func(gitops.CentralParams, gitops.Config) (v1alpha1.Central, error)

type cachedCentralRenderer struct {
	renderFn renderFn
	locks    *keyedMutex
	cache    *centralYamlCache
}

func newCachedCentralRenderer() *cachedCentralRenderer {
	return &cachedCentralRenderer{
		renderFn: gitops.RenderCentral,
		locks:    newKeyedMutex(),
		cache:    newCentralYamlCache(),
	}
}

func (r *cachedCentralRenderer) getCentralYaml(gitopsConfig gitops.Config, centralParams gitops.CentralParams) ([]byte, error) {
	centralID := centralParams.ID

	// We obtain a lock for the central ID so that no other goroutine can render the central yaml for the same central
	// at the same time.
	r.locks.lock(centralID)
	defer r.locks.unlock(centralID)

	// Computing the hash of both the gitops config and the incoming central params.
	// This should be a deterministic way to detect when the output of the render function would change.
	// In other words, the central YAML will always be the same for the same gitops config and central params.
	hashes, err := newHashes(gitopsConfig, centralParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get hash for Central")
	}

	var centralYaml []byte
	var shouldRender = true

	// Check if the central yaml is in the cache and if it is, check if the hash matches
	// If it does, we don't need to render it again
	r.cache.RLock()
	entry, ok := r.cache.entries[centralID]
	r.cache.RUnlock()
	if ok {
		if entry.hashes.equals(hashes) {
			// The hash matches, we can use the cached central yaml
			centralYaml = entry.centralYaml
			shouldRender = false
		}
	}

	if shouldRender {
		// There was no matching cache entry, we need to render the central yaml.
		centralCR, err := r.renderFn(centralParams, gitopsConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to apply GitOps overrides to Central")
		}
		centralYaml, err = yaml.Marshal(centralCR)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal Central CR")
		}
		// Locking the whole cache to add the new entry
		r.cache.Lock()
		defer r.cache.Unlock()
		r.cache.entries[centralID] = cacheEntry{
			id:          centralID,
			hashes:      hashes,
			centralYaml: centralYaml,
		}
	}

	return centralYaml, nil
}

type centralHashes struct {
	gitopsConfigHash  string
	centralParamsHash string
}

func (c centralHashes) equals(other centralHashes) bool {
	return c.gitopsConfigHash == other.gitopsConfigHash && c.centralParamsHash == other.centralParamsHash
}

// newHashes computes the hash of the gitops config and the central params
func newHashes(gitopsConfig gitops.Config, params gitops.CentralParams) (centralHashes, error) {
	h1, err := util.MD5SumFromJSONStruct(gitopsConfig)
	if err != nil {
		return centralHashes{}, errors.Wrap(err, "failed to get hash for GitOps config")
	}
	h2, err := util.MD5SumFromJSONStruct(params)
	if err != nil {
		return centralHashes{}, errors.Wrap(err, "failed to get hash for Central params")
	}
	return centralHashes{
		gitopsConfigHash:  fmt.Sprintf("%x", h1[:]),
		centralParamsHash: fmt.Sprintf("%x", h2[:]),
	}, nil
}

// keyedMutex is a mutex that can be locked by a key
type keyedMutex struct {
	locks map[string]*sync.Mutex
	mLock sync.RWMutex
}

func newKeyedMutex() *keyedMutex {
	return &keyedMutex{locks: make(map[string]*sync.Mutex)}
}

// getLockBy returns the lock for the given key. If the lock does not exist, it is created.
func (k *keyedMutex) getLockBy(key string) *sync.Mutex {
	k.mLock.RLock()
	if ret, ok := k.locks[key]; ok {
		k.mLock.RUnlock()
		return ret
	}
	k.mLock.RUnlock()
	k.mLock.Lock()
	defer k.mLock.Unlock()
	lock := &sync.Mutex{}
	k.locks[key] = lock
	return lock
}

// lock locks the lock for the given key
func (k *keyedMutex) lock(key string) {
	k.getLockBy(key).Lock()
}

// unlock unlocks the lock for the given key
func (k *keyedMutex) unlock(key string) {
	k.getLockBy(key).Unlock()
}

// tryLock tries to lock the lock for the given key
func (k *keyedMutex) tryLock(key string) bool {
	return k.getLockBy(key).TryLock()
}

// isLocked returns true if the lock for the given key is locked
func (k *keyedMutex) isLocked(key string) bool {
	tryLock := k.tryLock(key)
	if tryLock {
		k.unlock(key)
	}
	return !tryLock
}

// cacheEntry stores the central yaml and the hashes of the gitops config and the central params
type cacheEntry struct {
	id          string
	hashes      centralHashes
	centralYaml []byte
}

type centralYamlCache struct {
	sync.RWMutex
	entries map[string]cacheEntry
}

func newCentralYamlCache() *centralYamlCache {
	return &centralYamlCache{entries: make(map[string]cacheEntry)}
}
