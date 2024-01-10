# Gitops: configure ACS Central and ACS Operator


## Overview

Gitops provides a declarative configuration for ACS Central instances and ACS Operator.
GitOps relies on private repository to track changes to the configuration files.
[The repository](#gitops-repository) contains desired state for each environment configuration.


### Gitops repository

See: [Gitops repo](https://gitlab.cee.redhat.com/stackrox/acs-cloud-service/config)

### Gitops Flow

See: [detailed diagram](../../internal/dinosaur/pkg/gitops/README.md)


## Benefits

- **Consistency**: GitOps ensures consistency between configuration and state of the clusters on each environment.

- **Flexibility**: Gitops makes it possible to run multiple ACS operators simultaneously. It's also easy to override almost anything for the Central.

- **Validation**: GitOps repository CI validates each changes to spot configuration error as soon as possible.

- **Collaboration**: It's easy to make changes, review and then late merge it to the repository.

- **Auditability**: The versioned history of changes makes it easy to understand by whom, when and why changes were made.


## Canary rollout

See: [release instructions](https://gitlab.cee.redhat.com/stackrox/acs-cloud-service/config#release-rollout)
