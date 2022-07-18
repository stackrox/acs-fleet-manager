> NOTE this document refers to Red Hat internal components

# Deploying the Fleet Manager with AppInterface and Onboarding with AppSRE

To onboard with Service Delivery AppSRE, please follow the following [onboarding document](https://gitlab.cee.redhat.com/app-sre/contract/-/blob/master/content/service/service_onboarding_flow.md) which details the whole process.

## Envoy Configuration and Rate Limiting with AppInterface

All traffic goes through the Envoy container, which sends requests to the [3scale Limitador](https://github.com/Kuadrant/limitador) instance for global rate limiting across all API endpoints. If Limitador is unavailable, the response behaviour can be configured as desired. For this setup, the Envoy configuration when Limitador is unavailable is setup to forward all requests to the fleet-manager container. If the Envoy sidecar container itself was unavailable, the API would be unavailable.

1. The Envoy configuration for your deployment will need to be provided in AppInterface. Follow the [manage a config map](https://gitlab.cee.redhat.com/service/app-interface#example-manage-a-configmap-via-app-interface-openshiftnamespace-1yml) guideline.
The file [kas-fleet-manager Envoy ConfigMap](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/resources/services/managed-services/production/kas-fleet-manager-Envoy.configmap.yaml) can serve as a starting template for your fleet manager. This file is further referenced in [kas-fleet-manager namespace](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data/services/managed-services/namespaces/managed-services-production.yml#L58) so that the config map gets created.

2. Rate Limiting configuration is managed in [rate-limiting saas file](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data/services/rate-limiting/cicd/saas.yaml). Follow the [`kas-fleet-manager` example](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data/services/rate-limiting/cicd/saas.yaml#L197) to setup your configuration.

For more information on the setup, please see the [Rate Limiting template](https://gitlab.cee.redhat.com/service/rate-limiting-templates) and engage AppSRE for help.
