CHART_NAME = rhacs-terraform
OBSERVABILITY_SUBCHART_ROOT = charts/observability

.PHONY: helm/render
helm/render:
	helm -n ${ns} template $(CHART_NAME) . --debug -f ${values} ${helm_args}

.PHONY: helm/install
helm/install:
	helm -n ${ns} install $(CHART_NAME) . -f ${values} ${helm_args}

.PHONY: install
install: helm/install
	make -C $(OBSERVABILITY_SUBCHART_ROOT) helm/post-install ns=${ns}
	helm -n ${ns} list

.PHONY: uninstall
uninstall: # Example: make uninstall ns=rhacs
	helm -n ${ns} uninstall $(CHART_NAME)
	make -C $(OBSERVABILITY_SUBCHART_ROOT) helm/post-uninstall ns=${ns}
