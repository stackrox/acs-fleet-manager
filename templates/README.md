# Openshift Deployment Templates

The openshift deployment consists of 4 templates that, together, make an all-in-one deployment.

When deploying to production, the only template necessary is the service template.

## Service template

`templates/service-template.yml`

This is the main service template that deploys two objects, the `fleet-manager` deployment and the related service.

## Route template

`templates/route-template.yml`

This template just deploys a route with the select `app:fleet-manager` to map to the service deployed by the service template.

TLS is used by default for the route. No port is specified, all ports are allowed.

## Database template

`templates/db-template.yml`

This template deploys a simple postgresl-9.4 database deployment with a TLS-enabled service.

## Secrets template

`templates/secrets-template.yml`

This template deploys the `fleet-manager` secret with all of the necessary secret key/value pairs.

## Envoy Config template

`templates/envoy-config-template.yml`

This template deploys the `fleet-manager-envoy-config` ConfigMap that contains the Envoy
configuration for the `envoy-sidecar` container of the `fleet-manager` Deployment.