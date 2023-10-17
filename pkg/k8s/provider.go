// Package k8s ...
package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewKubernetesInterface returns a new Kubernetes interface or panics.
func NewKubernetesInterface() kubernetes.Interface {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	k8sInterface, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}

	return k8sInterface
}
