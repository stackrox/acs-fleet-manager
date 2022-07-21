# Quota Control
## Quota Management List Configurations

The type and the quantity of Standard Dinosaur instances a user can create is controlled via the
[Quota Management List](../config/quota-management-list-configuration.yaml).
If a user is not in the _Quota Management List_, only EVAL dinosaur instances will be allowed.

See [Allowing creation of *Standard* Dinosaur instances](populating-configuration.md#allowing-creation-of-standard-dinosaur-instances)
for details on differences between Standard and Eval instances.

### Adding organisations and users to the Quota Management List

#### With Red Hat based accounts
  To configure this list, you'll need to have the Red Hat account user's username
  and/or their organisation id.

  The username is the Red Hat account's username.

  The organization is the Red Hat account's organization ID.

  A practical way to get this information is by logging in to
  `cloud.redhat.com/openshift/token` with the account in question, then login
  to OCM with the OCM CLI and run the `ocm whoami` command to get information about that account.
  In the returned output:
  * `.username` is the Red Hat account's username
  * `.organization.external_id` is the Red Hat account's organization

### Max allowed instances
If the instance limit control is enabled, the service will enforce the `max_allowed_instances` configuration as the
limit to how many instances (i.e. Dinosaur) a user can create. This configuration can be specified per user or per
organisation in the quota list configuration. If not defined there, the service will take the default
`max_allowed_instances` into account instead.

Precedence of `max_allowed_instances` configuration: Org > User > Default.
