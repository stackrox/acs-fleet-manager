package reconciler

import (
	"context"
	"testing"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDeleteStaleTenant(t *testing.T) {
	keepNamespaces := []string{
		"keep-because-in-fm-list-1",
		"keep-because-in-fm-list-2",
		"keep-because-empty-tenant-name",
		"keep-because-not-managed-by-fleetshard",
		"keep-because-missing-tenant-label",
	}
	deleteNamespaces := []string{
		"delete-1",
		"delete-2",
	}

	existingObjs := []ctrlClient.Object{
		&corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: keepNamespaces[0],
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "rhacs-fleetshard",
					"rhacs.redhat.com/tenant":      keepNamespaces[0],
					"app.kubernetes.io/instance":   keepNamespaces[0],
				}},
		},
		&corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: deleteNamespaces[0],
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "rhacs-fleetshard",
					"rhacs.redhat.com/tenant":      deleteNamespaces[0],
					"app.kubernetes.io/instance":   deleteNamespaces[0],
				}},
		},
		&corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: keepNamespaces[1],
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "rhacs-fleetshard",
					"rhacs.redhat.com/tenant":      keepNamespaces[1],
					"app.kubernetes.io/instance":   keepNamespaces[1],
				}},
		},
		&corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: keepNamespaces[2],
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "rhacs-fleetshard",
					"rhacs.redhat.com/tenant":      keepNamespaces[2],
					"app.kubernetes.io/instance":   "",
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: deleteNamespaces[1],
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "rhacs-fleetshard",
					"rhacs.redhat.com/tenant":      deleteNamespaces[1],
					"app.kubernetes.io/instance":   deleteNamespaces[1],
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: keepNamespaces[3],
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "something-else",
					"rhacs.redhat.com/tenant":      keepNamespaces[3],
					"app.kubernetes.io/instance":   keepNamespaces[3],
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: keepNamespaces[4],
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "rhacs-fleetshard",
					"app.kubernetes.io/instance":   keepNamespaces[4],
				},
			},
		},
	}

	listFromFM := &private.ManagedCentralList{
		Items: []private.ManagedCentral{
			{
				Metadata: private.ManagedCentralAllOfMetadata{
					Namespace: keepNamespaces[0],
				},
			},
			{
				Metadata: private.ManagedCentralAllOfMetadata{
					Namespace: keepNamespaces[1],
				},
			},
		},
	}

	fakeClient, _ := testutils.NewFakeClientWithTracker(t, existingObjs...)
	tenantCleanup := NewTenantCleanup(fakeClient, NewTenantChartReconciler(fakeClient, true), NewNamespaceReconciler(fakeClient), NewCentralCrReconciler(fakeClient), true)
	err := tenantCleanup.DeleteStaleTenantK8sResources(context.TODO(), listFromFM)
	require.NoError(t, err, "unexpected error deleting stale tenants")

	namespaceList := corev1.NamespaceList{}
	err = fakeClient.List(context.TODO(), &namespaceList)
	require.NoError(t, err, "unexpected error listing remaining namespaces")

	remainingNames := make(map[string]bool, len(namespaceList.Items))
	for _, ns := range namespaceList.Items {
		remainingNames[ns.Name] = true
	}

	for _, keepNS := range keepNamespaces {
		if !remainingNames[keepNS] {
			t.Fatalf("expected namespace %q to still exist, but it was deleted", keepNS)
		}
	}

	for _, deleteNS := range deleteNamespaces {
		if remainingNames[deleteNS] {
			t.Fatalf("expected namespace %q to be deleted, but was not", deleteNS)
		}
	}

}
