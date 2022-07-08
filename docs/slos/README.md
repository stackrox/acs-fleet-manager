# Fleet Manager SLOs

Detailed information about SLOs can be found in Google's [Site Reliability Workbook](https://sre.google/workbook/table-of-contents/)

## Service Level Indicators
Service Level Indicators (SLIs) describe the basic properties of the metrics which are most relevant to the service provided.

The nature of the service should inform the choice of SLIs.
In general SLIs should be expressed as a ratio of two values and communicated as a percentage. Having a consistent SLI form allows for better alerts, analysis and reporting.

The majority of services would consider **availability** and **request latency** to be key SLIs.

**Availability** measures the proportion of requests that resulted in a successful response from the endpoints external users can interact with.

**Request latency** measures the proportion of requests served faster than a certain threshold.

SLIs should be reviewed and updated as the service develops.

For more detailed information about SLIs and practical examples of setting SLIs, read [Chapter 4 - Service Level Objectives](https://sre.google/sre-book/service-level-objectives/) of Site Reliability Workbook

## Service Level Objectives
Service Level Objectives (SLOs) are values or range of values to be delivered by the service. These targets are used to measure the performance of particular SLIs. SLOs should be defined after relevant SLIs are identified.

SLOs should be made in agreement with stakeholders and those commited to maintaining the service.

To decide on a starting SLO,  it is common to monitor the chosen SLIs over a period of time to identify the performance level of the service. The SLO can then be reviewed and updated as the service develops.

For more information about SLOs, please read [Chapter 4 - Service Level Objectives](https://sre.google/sre-book/service-level-objectives/) of Site Reliability Workbook.

## Alerts
Alerts are created to provide warnings when the service performance is outside of a particular threshold for a certain period of time. These alerts are used to ensure the service is meeting its SLOs.

Alerts are based on rules which provide the logic for when an alert fires. There are two critical concepts to formulating an alert rule; **error rate** and **error budget**.

**Error rate** is the ratio of error events to total events. This value can be used to provide the alerting threshold.
**Error budget** is the number of allowed error events as per defined SLO (for example an SLO of 99% means an error budget of 1% of total events over the time period). This value can be used to decide on the length of alert window.

A burn rate is a measure of how fast the service would consume the error budget relative to the SLO. For example:
With an SLO of 99% over a time period of 30 days, a constant 1% error rate uses exactly all of the error budget giving a burn rate of 1.

The alerts for this template follow a multi-window, multi-burn-rate structure. This is to ensure all significant error rates are alerted while also reducing the number of false positives. The section [Multiwindow, Multi-Burn-Rate Alerts](https://sre.google/workbook/alerting-on-slos/#:~:text=6%3A%20Multiwindow%2C%20Multi-Burn-Rate%20Alerts) of Site Reliability Workbook provides a detailed breakdown of this approach.

For more detailed information about alerts and practical examples, please read [Chapter 5 - Alerting on SLOs](https://sre.google/workbook/alerting-on-slos/) of Site Reliability Workbook.
