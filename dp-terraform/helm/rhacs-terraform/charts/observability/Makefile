CHART_NAME = rhacs-terraform-obs

.PHONY: helm/render
helm/render: # Example: make helm/render values=~/.rh/obs-values.yaml
	helm -n testn template $(CHART_NAME) . --debug -f ${values}

.PHONY: helm/install
helm/install: # Example: make helm/render ns=rhacs values=~/.rh/obs-values.yaml
	helm -n ${ns} install $(CHART_NAME) . -f ${values}

.PHONY: helm/post-install
helm/post-install:
	. hack/makerc.sh && wait_for_default_crd
	oc -n "${ns}-observability" delete observabilities.observability.redhat.com observability-stack
	oc -n "${ns}-observability" apply --filename hack/01-operator-06-observability-resource.yaml

.PHONY: install # Example: make install ns=rhacs values=~/.rh/obs-values.yaml
install: helm/install helm/post-install
	helm -n ${ns} list

.PHONY: uninstall

uninstall: # Example: make uninstall ns=rhacs
	helm -n ${ns} uninstall $(CHART_NAME)
# The Observability CRD is not correctly deleted. A workaround is removing the finalizer in the only Observability instance, and then deleting it
	oc -n "${ns}-observability" patch observabilities.observability.redhat.com rhacs-observability-stack --type json --patch='[ { "op": "remove", "path": "/metadata/finalizers" } ]' 
	oc delete customresourcedefinition observabilities.observability.redhat.com
