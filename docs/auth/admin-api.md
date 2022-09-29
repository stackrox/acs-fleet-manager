# AuthN/Z for the admin API of fleet manager

Related documents:
- [ADR on the middleware of the admin API](https://github.com/stackrox/architecture-decision-records/blob/eb0ac06392b9d36dd013079f7992423c9baa7238/managed_service/ADR-0011-fleet-manager-middleware-admin-api.md)

## Overview

The admin API gives administrative access to fleet manager. It includes the following functionality:
- Create / Update / Delete _all_ centrals within fleet manager, irrespective of ownership.
- Set specific resource requirements for central components, either during creation or within updates.

## Authentication

In contrast to the public and private API of fleet manager, which require a token issued by `https://sso.(env).redhat.com/auth/realms/redhat-external`,
the internal API requires tokens issued by the internal SSO: `https://auth.redhat.com/auth/realms/EmployeeIDP`.

This is only possible for internal Red Hat employees.

## Authorization

The access to the API is guarded by specific realm_access roles required for the API.
They are configured within the following files: [stage / dev config](../../config/admin/authz-roles-dev.yaml) and [prod config](../../config/admin-authz-roles-prod.yaml).

Internally, the roles are added by being a part of the corresponding group within Rover.

## How to call the API

1. Ensure you are a member of at least one of the rover groups that is configured.
2. Retrieve an auth token for the internal SSO. This is a bit more involved, so follow the steps below:
```bash

# Install the RHOAS CLI by visiting https://github.com/redhat-developer/app-services-cli or executing:
curl -o- https://raw.githubusercontent.com/redhat-developer/app-services-cli/main/scripts/install.sh | bash

# Run the login command and the correct auth URL. This will open a browser where you have to perform an interactive login
rhoas login --auth-url=https://auth.redhat.com/auth/realms/EmployeeIDP

# View the auth token
rhoas authtoken

# Export the auth token into a variable for later usage
token=$(rhoas authtoken)
```
3. Execute arbitrary API calls:
```bash
curl -H "Authorization: Bearer ${token}" http://fleet-manager:8000/api/rhacs/v1/admin/centrals
```
