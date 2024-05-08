package presenters

import (
	"context"
	"fmt"
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
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
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
	renderedCentral, err := c.renderer.render(gitopsConfig, centralParams)
	if err != nil {
		return private.ManagedCentral{}, errors.Wrap(err, "failed to get Central YAML")
	}
	authProvider := findAdditionalAuthProvider(gitopsConfig, from)
	additionalAuthProvider := constructAdditionalAuthProvider(authProvider, from)
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
			ExpiredAt:     dbapi.NullTimeToTimePtr(from.ExpiredAt),
		},
		Spec: private.ManagedCentralAllOfSpec{
			Owners: []string{
				from.Owner,
			},
			Auth: private.ManagedCentralAllOfSpecAuth{
				ClientId:             from.AuthConfig.ClientID,
				ClientSecret:         from.AuthConfig.ClientSecret, // pragma: allowlist secret
				ClientOrigin:         from.AuthConfig.ClientOrigin,
				OwnerOrgId:           from.OrganisationID,
				OwnerOrgName:         from.OrganisationName,
				OwnerUserId:          from.OwnerUserID,
				OwnerAlternateUserId: from.OwnerAlternateUserID,
				Issuer:               from.AuthConfig.Issuer,
			},
			AdditionalAuthProvider: additionalAuthProvider,
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
			CentralCRYAML:         renderedCentral.CentralCRYaml,
			TenantResourcesValues: renderedCentral.Values,
			InstanceType:          from.InstanceType,
		},
		RequestStatus: from.Status,
	}

	if from.DeletionTimestamp.Valid {
		res.Metadata.DeletionTimestamp = from.DeletionTimestamp.Time.Format(time.RFC3339)
	}

	return res, nil
}

func constructAdditionalAuthProvider(authProvider *gitops.AuthProvider, from *dbapi.CentralRequest) private.ManagedCentralAllOfSpecAdditionalAuthProvider {
	if authProvider == nil {
		return private.ManagedCentralAllOfSpecAdditionalAuthProvider{}
	}
	oidcConfig := constructAdditionalOidcConfig(authProvider, from)

	groups := make([]private.ManagedCentralAllOfSpecAdditionalAuthProviderGroups, 0, len(authProvider.Groups))
	for _, group := range authProvider.Groups {
		groups = append(groups, private.ManagedCentralAllOfSpecAdditionalAuthProviderGroups{
			Key:   group.Key,
			Value: group.Value,
			Role:  group.Role,
		})
	}

	requiredAttributes := make([]private.ManagedCentralAllOfSpecAdditionalAuthProviderRequiredAttributes, 0, len(authProvider.RequiredAttributes))
	for _, requiredAttribute := range authProvider.RequiredAttributes {
		requiredAttributes = append(requiredAttributes, private.ManagedCentralAllOfSpecAdditionalAuthProviderRequiredAttributes{
			Key:   requiredAttribute.Key,
			Value: requiredAttribute.Value,
		})
	}

	claimMappings := make([]private.ManagedCentralAllOfSpecAdditionalAuthProviderRequiredAttributes, 0, len(authProvider.ClaimMappings))
	for _, claimMapping := range authProvider.ClaimMappings {
		claimMappings = append(claimMappings, private.ManagedCentralAllOfSpecAdditionalAuthProviderRequiredAttributes{
			Key:   claimMapping.Path,
			Value: claimMapping.Name,
		})
	}
	return private.ManagedCentralAllOfSpecAdditionalAuthProvider{
		Name:               authProvider.Name,
		MinimumRoleName:    authProvider.MinimumRole,
		Groups:             groups,
		RequiredAttributes: requiredAttributes,
		ClaimMappings:      claimMappings,
		Oidc:               oidcConfig,
	}
}

func constructAdditionalOidcConfig(authProvider *gitops.AuthProvider, from *dbapi.CentralRequest) private.ManagedCentralAllOfSpecAdditionalAuthProviderOidc {
	oidcConfig := private.ManagedCentralAllOfSpecAdditionalAuthProviderOidc{}
	if authProvider.OIDC != nil && authProvider.OIDC.ClientID != "" {
		oidcConfig.ClientID = authProvider.OIDC.ClientID
		oidcConfig.ClientSecret = authProvider.OIDC.ClientSecret
		oidcConfig.Issuer = authProvider.OIDC.Issuer
		oidcConfig.CallbackMode = authProvider.OIDC.Mode
		oidcConfig.DisableOfflineAccessScope = authProvider.OIDC.DisableOfflineAccessScope
	} else {
		oidcConfig.ClientID = from.ClientID
		oidcConfig.ClientSecret = from.ClientSecret
		oidcConfig.Issuer = from.Issuer
		oidcConfig.CallbackMode = "post"
		oidcConfig.DisableOfflineAccessScope = true
	}
	return oidcConfig
}

func findAdditionalAuthProvider(gitopsConfig gitops.Config, from *dbapi.CentralRequest) *gitops.AuthProvider {
	for _, addition := range gitopsConfig.Centrals.AdditionalAuthProviders {
		if addition.InstanceID == from.ID {
			return addition.AuthProvider
		}
	}
	return nil
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

type renderCentralFn func(gitops.CentralParams, gitops.Config) (v1alpha1.Central, error)
type renderValuesFn func(gitops.CentralParams, gitops.Config) (map[string]interface{}, error)

type cachedCentralRenderer struct {
	renderCentralFn renderCentralFn
	renderValuesFn  renderValuesFn
	locks           *keyedMutex
	cache           *centralYamlCache
}

func newCachedCentralRenderer() *cachedCentralRenderer {
	return &cachedCentralRenderer{
		renderCentralFn: gitops.RenderCentral,
		renderValuesFn:  gitops.RenderTenantResourceValues,
		locks:           newKeyedMutex(),
		cache:           newCentralYamlCache(),
	}
}

type RenderedCentral struct {
	CentralCRYaml string
	Values        map[string]interface{}
}

func (r *cachedCentralRenderer) render(gitopsConfig gitops.Config, centralParams gitops.CentralParams) (RenderedCentral, error) {
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
		return RenderedCentral{}, errors.Wrap(err, "failed to get hash for Central")
	}

	var centralYaml []byte
	var values map[string]interface{}
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
			values = entry.values
			shouldRender = false
		}
	}

	if shouldRender {
		// There was no matching cache entry, we need to render the central yaml.
		centralCR, err := r.renderCentralFn(centralParams, gitopsConfig)
		if err != nil {
			return RenderedCentral{}, errors.Wrap(err, "failed to apply GitOps overrides to Central")
		}
		centralYaml, err = yaml.Marshal(centralCR)
		if err != nil {
			return RenderedCentral{}, errors.Wrap(err, "failed to marshal Central CR")
		}
		values, err := r.renderValuesFn(centralParams, gitopsConfig)
		if err != nil {
			return RenderedCentral{}, errors.Wrap(err, "failed to render tenant resource values")
		}
		// Locking the whole cache to add the new entry
		r.cache.Lock()
		defer r.cache.Unlock()
		r.cache.entries[centralID] = cacheEntry{
			id:          centralID,
			hashes:      hashes,
			centralYaml: centralYaml,
			values:      values,
		}
	}

	return RenderedCentral{
		CentralCRYaml: string(centralYaml),
		Values:        values,
	}, nil
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
	mLock sync.Mutex
}

func newKeyedMutex() *keyedMutex {
	return &keyedMutex{locks: make(map[string]*sync.Mutex)}
}

// getLockBy returns the lock for the given key. If the lock does not exist, it is created.
func (k *keyedMutex) getLockBy(key string) *sync.Mutex {
	k.mLock.Lock()
	defer k.mLock.Unlock()
	if ret, ok := k.locks[key]; ok {
		return ret
	}
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
	values      map[string]interface{}
}

type centralYamlCache struct {
	sync.RWMutex
	entries map[string]cacheEntry
}

func newCentralYamlCache() *centralYamlCache {
	return &centralYamlCache{entries: make(map[string]cacheEntry)}
}
