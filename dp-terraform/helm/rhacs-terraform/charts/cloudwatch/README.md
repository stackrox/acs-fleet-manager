# Data plane terraform cloudwatch exporter Helm chart

## Configuration

The [cloudwatch exporter](https://github.com/prometheus/cloudwatch_exporter) is configured via the
`cloudwatch-exporter-config` config map. See the [AWS metrics list](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/Aurora.AuroraMySQL.Monitoring.Metrics.html) and [AWS serverless capacity guide](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora-serverless-v2.setting-capacity.html) for possible database metric series to export.

## Authentication

The `rhacs-cloudwatch-exporter` secret must contain AWS credentials with the following permissions:

```
"cloudwatch:GetMetricData",
"cloudwatch:GetMetricStatistics",
"cloudwatch:ListMetrics",
"tag:GetResources",
```
