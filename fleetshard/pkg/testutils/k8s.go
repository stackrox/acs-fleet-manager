package testutils

import (
	"testing"

	openshiftRouteV1 "github.com/openshift/api/route/v1"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8sTesting "k8s.io/client-go/testing"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	centralsGVR = schema.GroupVersionResource{
		Group:    "platform.stackrox.io",
		Version:  "v1alpha1",
		Resource: "centrals",
	}
	secretsGVR = schema.GroupVersionResource{
		Version:  "v1",
		Resource: "secrets",
	}
)

// CentralCA ...
const CentralCA = "test CA"

type reconcileTracker struct {
	k8sTesting.ObjectTracker
}

// NewFakeClientBuilder returns a new fake client builder with registered custom resources
func NewFakeClientBuilder(t *testing.T, objects ...ctrlClient.Object) *fake.ClientBuilder {
	scheme := runtime.NewScheme()
	require.NoError(t, platform.AddToScheme(scheme))
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, openshiftRouteV1.Install(scheme))

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjectTracker(newReconcileTracker(scheme)).
		WithObjects(objects...)
}

func newReconcileTracker(scheme *runtime.Scheme) k8sTesting.ObjectTracker {
	return reconcileTracker{ObjectTracker: k8sTesting.NewObjectTracker(scheme, clientgoscheme.Codecs.UniversalDecoder())}
}

// Create ...
func (t reconcileTracker) Create(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	if err := t.ObjectTracker.Create(gvr, obj, ns); err != nil {
		return err
	}
	if gvr == centralsGVR {
		centralTlsSecret := &coreV1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "central-tls",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"ca.pem": []byte(CentralCA),
			},
		}
		return t.ObjectTracker.Create(secretsGVR, centralTlsSecret, ns)
	}
	return nil
}
