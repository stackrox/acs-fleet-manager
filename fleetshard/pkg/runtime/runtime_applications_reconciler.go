package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	argocd "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/golang/glog"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/argox"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
)

type runtimeApplicationsReconciler struct {
	client                   ctrlClient.Client
	lastAppliedConfiguration []byte
	namespace                string
}

func newRuntimeApplicationsReconciler(client ctrlClient.Client, namespace string) *runtimeApplicationsReconciler {
	return &runtimeApplicationsReconciler{
		client:    client,
		namespace: namespace,
	}
}

func (r *runtimeApplicationsReconciler) reconcile(ctx context.Context, cfg private.ManagedCentralList) error {

	jsonBytes, err := json.Marshal(cfg.Applications)
	if err != nil {
		return err
	}

	if bytes.Equal(jsonBytes, r.lastAppliedConfiguration) {
		glog.V(10).Info("runtime argocd applications configuration has not changed, skipping reconciliation")
		return nil
	}

	var applicationList []*argocd.Application
	for i, val := range cfg.Applications {
		appJson, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("failed to marshal application %d: %w", i, err)
		}

		app := &argocd.Application{}
		if err := json.Unmarshal(appJson, app); err != nil {
			return fmt.Errorf("failed to unmarshal application %d: %w", i, err)
		}

		applicationList = append(applicationList, app)
	}

	// !Do not change this without some sort of migration.
	selector := map[string]string{
		"app.kubernetes.io/managed-by": "rhacs-fleetshard",
		"rhacs.redhat.com/part-of":     "runtime-applications",
	}

	err = argox.ReconcileApplications(ctx, r.client, r.namespace, selector, applicationList)
	if err != nil {
		return fmt.Errorf("failed to reconcile applications: %w", err)
	}

	return nil

}
