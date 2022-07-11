# Fleet Manager API - Latency SLO/SLI

## SLI description
We are measuring the proportion of requests served faster than a certain threshold.

## SLI Rationale
The Fleet-manager API is a critical component in any service ecosystem, it is expected to provide sufficiently fast responses to ensure good user experience.

## Implementation details
There are two SLIs backing these two SLOs. Both use the same metric with a different request duration value. We use the `api_inbound_request_duration_bucket` histogram metric as the base of this SLO.

Since this metric is shared with other managed services, it needs labels for the fleet-manager to filter the results to fleet-manager, `job="fleet-manager-metrics",namespace="managed-services-production"`. The implementation is also only including successsful responses, so the code label is added `,code!~"5.."`.

An appropriate SLI implementation is the count of successful API HTTP requests within a certain duration divided by the count of all of API HTTP requests.

## SLO Rationale
The SLI implementation should be chosen based on observing the service running in production over a long period of time and from running API performance tests.

## Alerts

> NOTE this section contains references to Red Hat internal components

All alerts are multiwindow, multi-burn-rate alerts. The following are the list of alerts that are associated with this SLO.

- `FleetManagerAPILatency30mto6hP99BudgetBurn`
- `FleetManagerAPILatency2hto1dor6hto3dP99BudgetBurn`
- `FleetManagerAPILatency30mto6hP90BudgetBurn`
- `FleetManagerAPILatency2hto1dor6hto3dP90BudgetBurn`

See [kas-fleet-manager-slos-latency-*](https://gitlab.cee.redhat.com/service/app-interface/-/tree/master/resources/observability/prometheusrules) prometheus rules in AppInterface to see how it was implemented.
