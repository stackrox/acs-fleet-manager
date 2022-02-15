> NOTE this document refers to Red Hat internal components

# Quota Management with Account Management Service (AMS) SKU

The [Account Management Service](https://api.openshift.com/?urls.primaryName=Accounts%20management%20service) manages users subscriptions. The fleet manager uses the service offered by AMS to manage quota. Quota comes in form of stock keeping unit (SKU) assigned to a given organisation. The process is the same as requesting SKU for addons which is described in the following link https://gitlab.cee.redhat.com/service/managed-tenants/-/blob/main/docs/tenants/requesting_skus.md.

# User Account & Organization Setup
1. Create new organization and users being members of this organization:
* Make sure that you are logged out from any existing `access.redhat.com` sessions. Use the `register new organization` flow to create an organization
* Create new users using user management UI. It is recommended to append or prefix usernames of your team members, e.g. `dffrench_control_plane` for `dffrench` username being a member of `Control Plane` team.

2. To onboard a fleet-manager to [app-interface](https://gitlab.cee.redhat.com/service/app-interface) a new role will be required for resources (e.g. Syncsets, etc) creation (see [example](https://gitlab.cee.redhat.com/service/uhc-account-manager/-/blob/master/pkg/api/roles/managed_kafka_service.go)). The following [MR](https://gitlab.cee.redhat.com/service/uhc-account-manager/-/merge_requests/1907) shows how the ManagedKafkaService role was created for RHOSAK.
3. Once the role is created, users required to have this role need to get it assigned to their OCM account: [example MR](https://gitlab.cee.redhat.com/service/ocm-resources/-/merge_requests/812).
4. Very likely your organization will need to have quota for installing Add-ons specific for your fleet-manager. See an example of Add-on quotas [here](https://gitlab.cee.redhat.com/service/ocm-resources/-/blob/master/data/uhc-stage/orgs/13640203.yaml) (find your organization by its `external_id` beneath [ocm-resources/uhc-stage/orgs](https://gitlab.cee.redhat.com/service/ocm-resources/-/tree/master/data/uhc-stage/orgs)).
5. Each call to OCM endpoints needs specific permissions. Make sure that you have all the needed permissions to call OCM endpoints. If the role created in step (2) is missing those permissions, then they can be added via the [account-manager repo](https://gitlab.cee.redhat.com/service/uhc-account-manager). See an example [MR](https://gitlab.cee.redhat.com/service/uhc-account-manager/-/merge_requests/2530). 
   >TIP: The permissions attached to the [`ManagedKafkaService`](https://gitlab.cee.redhat.com/service/uhc-account-manager/-/blob/master/pkg/api/roles/managed_kafka_service.go) role will be a very good start.

# Vault
For automation (fleet-manager deployments and app-interface pipelines) it is required to use [vault](https://gitlab.cee.redhat.com/service/app-interface#manage-secrets-via-app-interface-openshiftnamespace-1yml-using-vault) for secrets management

# AWS
AWS accounts are required for provisioning development OSD clusters and to access AWS Route53 for Domain Name System. To request a AWS accounts, have a chat with your team's manager.  
> NOTE, if you are in the Middleware, make sure to send an email to `mw-billing-leaders@redhat.com` request the account to be created. The email should follow the below format:
```
*Request Type*
New AWS account
*Team*
<Whoever the requesting team is>
 
*Cost*
<Calculations for the estimated monthly charges>
*Why?*
<The reason for requesting the account> 
```
> NOTE
Within the AWS account used to provision OSD clusters, you must create an osdCcsAdmin IAM user with the following requirements:
- This user needs at least Programmatic access enabled.
- This user must have the AdministratorAccess policy attached to it.
See https://docs.openshift.com/dedicated/osd_planning/aws-ccs.html#ccs-aws-customer-procedure_aws-ccs for more information.

# SSO authentication
Depending on whether interacting with public or private endpoints, the authentication should be set up as follows:
* sso.redhat.com for public endpoints
* MAS-SSO for private endpoints (see this [doc](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/docs/mas-sso/service-architecture/service-architecture.md) to understand how kas-fleet-manager is using MAS-SSO for authentication)

See [feature-flags](feature-flags.md#Keycloak) to understand flags used for authentication.

## sso.redhat.com service account
To avail of all required OCM services the fleet-manager depends on, it is required to create an sso.redhat.com service account that will be used for the communication between the fleet-manager and OCM. It is a help desk ticket to get this created which gets routed to Red Hat IT. 
The link to create the request is https://redhat.service-now.com/help?id=sc_cat_item&sys_id=7ab45993131c9380196f7e276144b054
