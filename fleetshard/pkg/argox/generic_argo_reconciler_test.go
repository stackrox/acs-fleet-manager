package argox

import (
	"context"
	"testing"

	argocd "github.com/stackrox/acs-fleet-manager/pkg/argocd/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func Test_reconcileArgoCDApplications_deletesUnwantedApplications(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-1",
					Namespace: "my-namespace",
					Labels:    map[string]string{"key": "value"},
				},
			},
		).
		Build()

	require.NoError(t, ReconcileApplications(context.Background(), cli, "my-namespace", map[string]string{"key": "value"}, nil))

	var appList argocd.ApplicationList
	require.NoError(t, cli.List(context.Background(), &appList, client.InNamespace("my-namespace")))
	assert.Empty(t, appList.Items)

}

func Test_reconcileArgoCDApplications_doesNotDeleteNonMatchingApplications(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			// This application should not be deleted because its selector is different
			&argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app",
					Namespace: "my-namespace",
					Labels: map[string]string{
						"should": "be-ignored",
					},
				},
			},
		).
		Build()

	require.NoError(t, ReconcileApplications(context.Background(), cli, "my-namespace", map[string]string{"key": "value"}, nil))

	var appList argocd.ApplicationList
	require.NoError(t, cli.List(context.Background(), &appList, client.InNamespace("my-namespace")))
	require.Len(t, appList.Items, 1)
	assert.Equal(t, "my-app", appList.Items[0].Name)

}

func Test_reconcileArgoCDApplications_createsMissingApplications(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	require.NoError(t, ReconcileApplications(context.Background(), cli, "my-namespace", map[string]string{"key": "value"}, []*argocd.Application{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-app",
			},
		},
	}))

	var appList argocd.ApplicationList
	require.NoError(t, cli.List(context.Background(), &appList, client.InNamespace("my-namespace")))
	require.Len(t, appList.Items, 1)
	assert.Equal(t, "my-app", appList.Items[0].Name)
	assert.Equal(t, "my-namespace", appList.Items[0].Namespace)                 // Ensures that the reconciler sets the namespace
	assert.Equal(t, map[string]string{"key": "value"}, appList.Items[0].Labels) // Ensures that the reconciler sets the labels to match the selector

}

func Test_reconcileArgoCDApplications_updatesExistingApplications(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&argocd.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "my-namespace",
				Labels:    map[string]string{"key": "value"},
			},
		}).
		Build()

	require.NoError(t, ReconcileApplications(context.Background(), cli, "my-namespace", map[string]string{"key": "value"}, []*argocd.Application{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-app",
			},
		},
	}))

	var appList argocd.ApplicationList
	require.NoError(t, cli.List(context.Background(), &appList, client.InNamespace("my-namespace")))
	require.Len(t, appList.Items, 1)
	assert.Equal(t, "my-app", appList.Items[0].Name)
	assert.NotEmpty(t, appList.Items[0].Annotations["rhacs.redhat.com/last-applied-configuration"]) // Ensures that the reconciler sets the last-applied-configuration annotation

}

func Test_reconcileArgoCDApplications_updatesExistingApplications_withDifferentHash(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&argocd.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "my-namespace",
				Labels:    map[string]string{"key": "value"},
			},
		}).
		Build()

	require.NoError(t, ReconcileApplications(context.Background(), cli, "my-namespace", map[string]string{"key": "value"}, []*argocd.Application{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-app",
				Annotations: map[string]string{
					"rhacs.redhat.com/last-applied-configuration": "different-hash",
				},
			},
		},
	}))

	var appList argocd.ApplicationList
	require.NoError(t, cli.List(context.Background(), &appList, client.InNamespace("my-namespace")))
	require.Len(t, appList.Items, 1)
	assert.Equal(t, "my-app", appList.Items[0].Name)
	assert.NotEqual(t, "different-hash", appList.Items[0].Annotations["rhacs.redhat.com/last-applied-configuration"]) // Ensures that the reconciler updates the application

}

func Test_reconcileArgoCDApplications_doesNotUpdate_ifHashMatch(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	desired := &argocd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-app",
		},
	}
	jsonBytes, err := getJsonString(desired)
	require.NoError(t, err)

	updateCalled := false

	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&argocd.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "my-namespace",
				Labels:    map[string]string{"key": "value"},
				Annotations: map[string]string{
					"rhacs.redhat.com/last-applied-configuration": string(jsonBytes),
				},
			},
		}).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				updateCalled = true
				return client.Update(ctx, obj, opts...)
			},
		}).
		Build()

	require.NoError(t, ReconcileApplications(context.Background(), cli, "my-namespace", map[string]string{"key": "value"}, []*argocd.Application{
		desired,
	}))

	assert.False(t, updateCalled)

}
