# Emailsender Helm Chart

This Helm chart deploys the emailsender service for RHACS dataplane clusters.

## Overview

The emailsender service handles email notifications for RHACS tenants. It uses AWS SES as the email provider and requires a PostgreSQL database for storing email records.

## Prerequisites

- Kubernetes cluster with OpenShift service-ca operator (for HTTPS support)
- External Secrets Operator (for AWS secrets management)
- AWS IAM role for SES access

## Configuration

See [values.yaml](values.yaml) for the full list of configuration options.

### Key Configuration Values

- `replicas`: Number of replicas (default: 3)
- `image.repo`: Container image repository
- `image.tag`: Container image tag
- `clusterId`: Data plane cluster ID
- `clusterName`: Data plane cluster name
- `environment`: Environment name (e.g., "production", "staging")
- `senderAddress`: Email sender address
- `emailProvider`: Email provider (default: "AWS_SES")
- `aws.region`: AWS region for SES

## Installation

```bash
helm install emailsender ./deploy/charts/emailsender \
  --set clusterId=my-cluster \
  --set clusterName=my-cluster \
  --set environment=production \
  --set aws.region=us-east-1
```

## Components

The chart deploys:

1. **Deployment**: The emailsender service with 3 replicas by default
2. **Service**: ClusterIP service exposing port 443 (HTTPS)
3. **ServiceAccount**: For AWS IAM role integration
4. **RBAC**: ClusterRole and ClusterRoleBinding
5. **ExternalSecrets**: For database credentials and AWS role ARN

## Database

The emailsender requires a PostgreSQL database. Database credentials are managed via External Secrets Operator and stored in AWS Secrets Manager.

## HTTPS/TLS

The service uses OpenShift's service-ca operator to generate TLS certificates. This can be disabled by setting `enableHTTPS=false` for clusters without the service-ca operator.
