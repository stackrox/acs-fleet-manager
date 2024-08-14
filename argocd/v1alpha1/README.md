These types were copy-pasted from https://github.com/argoproj/argo-cd/tree/master/pkg/apis/application/v1alpha1

We are not importing the ArgoCD dependency itself, because it would significantly
complexify the go.mod, and might introduce conflicts with the stackrox dependencies.
