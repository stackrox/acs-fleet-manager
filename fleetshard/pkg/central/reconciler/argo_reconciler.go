package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/argox"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	argocd "github.com/stackrox/acs-fleet-manager/pkg/argocd/apis/application/v1alpha1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type argoReconciler struct {
	client   ctrlClient.Client
	argoOpts ArgoReconcilerOptions
}

// ArgoReconcilerOptions defines configuration options for the Argo application reconciliation
type ArgoReconcilerOptions struct {
	TenantDefaultArgoCdAppSourceRepoURL        string
	TenantDefaultArgoCdAppSourceTargetRevision string
	TenantDefaultArgoCdAppSourcePath           string
	ArgoCdNamespace                            string
	ManagedDBEnabled                           bool
	ClusterName                                string
	Environment                                string
	WantsAuthProvider                          bool
	Telemetry                                  config.Telemetry
}

func newArgoReconciler(
	client ctrlClient.Client,
	argoReconcilerOptions ArgoReconcilerOptions,
) *argoReconciler {
	return &argoReconciler{
		client:   client,
		argoOpts: argoReconcilerOptions,
	}
}

func (r *argoReconciler) ensureApplicationExists(ctx context.Context, remoteCentral private.ManagedCentral, centralDBConnectionString string) error {
	want, err := r.makeDesiredArgoCDApplication(remoteCentral, centralDBConnectionString)
	if err != nil {
		return fmt.Errorf("getting ArgoCD application: %w", err)
	}
	if err := argox.ReconcileApplication(ctx, r.client, want); err != nil {
		return fmt.Errorf("reconciling ArgoCD application: %w", err)
	}
	return nil
}

func (r *argoReconciler) makeDesiredArgoCDApplication(remoteCentral private.ManagedCentral, centralDBConnectionString string) (*argocd.Application, error) {

	values := remoteCentral.Spec.TenantResourcesValues
	if values == nil {
		values = map[string]interface{}{}
	}

	// Invariants
	values["environment"] = r.argoOpts.Environment
	values["clusterName"] = r.argoOpts.ClusterName
	values["organizationId"] = remoteCentral.Spec.Auth.OwnerOrgId
	values["organizationName"] = remoteCentral.Spec.Auth.OwnerOrgName
	values["instanceId"] = remoteCentral.Id
	values["instanceName"] = remoteCentral.Metadata.Name
	values["instanceType"] = remoteCentral.Spec.InstanceType
	values["isInternal"] = remoteCentral.Metadata.Internal
	values["telemetryStorageEndpoint"] = r.argoOpts.Telemetry.StorageEndpoint
	values["centralAdminPasswordEnabled"] = !r.argoOpts.WantsAuthProvider
	values["centralUIHost"] = remoteCentral.Spec.UiHost
	values["centralDataHost"] = remoteCentral.Spec.DataHost

	if remoteCentral.Metadata.ExpiredAt != nil {
		values["expiredAt"] = remoteCentral.Metadata.ExpiredAt.Format(time.RFC3339)
	} else {
		values["expiredAt"] = ""
	}

	if !remoteCentral.Metadata.Internal && r.argoOpts.Telemetry.StorageKey != "" {
		values["telemetryStorageKey"] = r.argoOpts.Telemetry.StorageKey
	} else {
		values["telemetryStorageKey"] = "DISABLED"
	}

	if r.argoOpts.ManagedDBEnabled {
		values["centralDbSecretName"] = centralDbSecretName // pragma: allowlist secret
		values["centralDbConnectionString"] = centralDBConnectionString

		dbCA, err := postgres.GetDatabaseCACertificates()
		if err != nil {
			glog.Warningf("Could not read DB server CA bundle: %v", err)
		} else {
			values["additionalCAs"] = []map[string]interface{}{
				{
					"name":    postgres.CentralDatabaseCACertificateBaseName,
					"content": string(dbCA),
				},
			}
		}
	} else {
		delete(values, "centralDbSecretName")
		delete(values, "centralDbConnectionString")
		delete(values, "additionalCAs")
	}

	if isArgoDeclarativeConfigReconciliationEnabled(remoteCentral) {
		dc, _ := values["declarativeConfig"].(map[string]interface{})
		if dc == nil {
			dc = map[string]interface{}{}
			values["declarativeConfig"] = dc
		}
		dc["defaultAuthProvider"] = map[string]interface{}{
			"oidc": map[string]interface{}{
				"issuer":                  remoteCentral.Spec.Auth.Issuer,
				"clientCredentialsSecret": authProviderClientCredentialsSecretName,
			},
			"ownerUserId":          remoteCentral.Spec.Auth.OwnerUserId,
			"ownerAlternateUserId": remoteCentral.Spec.Auth.OwnerAlternateUserId,
		}
	}

	valuesBytes, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshalling values: %w", err)
	}

	return &argocd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      remoteCentral.Metadata.Namespace,
			Namespace: r.getArgoCdAppNamespace(),
		},
		Spec: argocd.ApplicationSpec{
			Project: "default",
			SyncPolicy: &argocd.SyncPolicy{
				Automated: &argocd.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
			},
			Source: &argocd.ApplicationSource{
				RepoURL:        r.getSourceRepoURL(remoteCentral),
				TargetRevision: r.getSourceTargetRevision(remoteCentral),
				Path:           r.getSourcePath(remoteCentral),
				Helm: &argocd.ApplicationSourceHelm{
					ValuesObject: &runtime.RawExtension{
						Raw: valuesBytes,
					},
				},
			},
			Destination: argocd.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: remoteCentral.Metadata.Namespace,
			},
		},
	}, nil
}

func (r *argoReconciler) getSourceTargetRevision(m private.ManagedCentral) string {
	return getTenantResourcesValue(m, "argoCd.sourceTargetRevision", r.argoOpts.TenantDefaultArgoCdAppSourceTargetRevision)
}

func (r *argoReconciler) getSourcePath(m private.ManagedCentral) string {
	return getTenantResourcesValue(m, "argoCd.sourcePath", r.argoOpts.TenantDefaultArgoCdAppSourcePath)
}

func (r *argoReconciler) getSourceRepoURL(m private.ManagedCentral) string {
	return getTenantResourcesValue(m, "argoCd.sourceRepoUrl", r.argoOpts.TenantDefaultArgoCdAppSourceRepoURL)
}

func isArgoDeclarativeConfigReconciliationEnabled(m private.ManagedCentral) bool {
	return getTenantResourcesValue(m, "declarativeConfig.enabled", false)
}

func isForceReconcile(m private.ManagedCentral) bool {
	return getTenantResourcesValue(m, "forceReconcile", false)
}

type valueType interface {
	string | bool | int | float64
}

func getTenantResourcesValue[T valueType](remoteCentral private.ManagedCentral, path string, defaultValue T) T {
	return getHelmValueByPath(remoteCentral.Spec.TenantResourcesValues, path, defaultValue)
}

func getHelmValueByPath[T valueType](values map[string]interface{}, path string, defaultValue T) T {
	if values == nil {
		return defaultValue
	}

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return defaultValue
	}

	current := values
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]interface{})
		if !ok {
			return defaultValue
		}
		current = next
	}

	val, ok := current[parts[len(parts)-1]].(T)
	if !ok {
		return defaultValue
	}

	return val
}

func (r *argoReconciler) ensureApplicationDeleted(ctx context.Context, tenantNamespace string) (bool, error) {
	app := &argocd.Application{}
	objectKey := r.getArgoCdAppObjectKey(tenantNamespace)

	err := wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		err := r.client.Get(ctx, objectKey, app)
		if apiErrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, fmt.Errorf("getting ArgoCD application: %w", err)
		}

		if app.DeletionTimestamp != nil {
			return false, nil
		}

		if err := r.client.Delete(ctx, app); err != nil {
			return false, fmt.Errorf("deleting ArgoCD application: %w", err)
		}

		return false, nil
	})
	if err != nil {
		return false, fmt.Errorf("waiting for ArgoCD application deletion: %w", err)
	}

	return true, nil
}

func (r *argoReconciler) getArgoCdAppNamespace() string {
	return r.argoOpts.ArgoCdNamespace
}

func (r *argoReconciler) getArgoCdAppObjectKey(tenantNamespace string) ctrlClient.ObjectKey {
	return ctrlClient.ObjectKey{
		Namespace: r.getArgoCdAppNamespace(),
		Name:      tenantNamespace,
	}
}
