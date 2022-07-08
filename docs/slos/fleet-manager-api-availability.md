# Fleet Manager API - Availability SLO/SLI

## SLI description
We are measuring the proportion of requests that resulted in a successful response from the endpoints external users can interact with.

## SLI Rationale
The Fleet-manager API is a critical component in any service ecosystem, it is expected to be available and responding successfully to requests.

## Implementation details
We count the number of API requests that do not have a `5xx` status code and divide it by the total of all the API requests made.
It is measured at the router using the `haproxy_backend_http_responses_total` metric.

## SLO Rationale
A Fleet-manager should be available 95 percent of the time. This could be increased once this availability SLO has been proven in production over a longer period of time.

## Alerts
All alerts are multiwindow, multi-burn-rate alerts. The following are the list of alerts that are associated with this SLO.

- `FleetManagerAPI30mto6hErrorBudgetBurn`
- `FleetManagerAPI2hto1dErrorBudgetBurn`
- `FleetManagerAPI6hto3dErrorBudgetBurn`

See [kas-fleet-manager-slos-availability-*](https://gitlab.cee.redhat.com/service/app-interface/-/tree/master/resources/observability/prometheusrules) prometheus rules in AppInterface to see how it was implemented.