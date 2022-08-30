## OpenShift Router Deployment Manifests

This directory contains the following manifests:

1. `00-apps-openshift-dummy.crd.yaml`: A dummy CRD for convincing the RHACS Operator that it is running on OpenShift so that creates OpenShift routes.
1. `01-router_rbac.yaml`: Downloaded from [https://raw.githubusercontent.com/openshift/router/master/deploy/router_rbac.yaml](https://raw.githubusercontent.com/openshift/router/master/deploy/router_rbac.yaml)
1. `02-route.crd.yaml`: Downloaded from [https://raw.githubusercontent.com/openshift/api/master/route/v1/route.crd.yaml](https://raw.githubusercontent.com/openshift/api/master/route/v1/route.crd.yaml).
1. `03-router.yaml`: Downloaded from [https://raw.githubusercontent.com/openshift/router/master/deploy/router.yaml](https://raw.githubusercontent.com/openshift/router/master/deploy/router.yaml) and modified by adding `{"name": "ROUTER_CANONICAL_HOSTNAME", "value": "$CLUSTER_DNS"}` to `spec.template.spec.containers[?(@.name == "router")].env`

All the files have been downloaded on August 30, 2022.
