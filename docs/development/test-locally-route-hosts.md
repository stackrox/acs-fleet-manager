# How to access hosts created by Routes locally

When the Routes are created locally on vanilla k8s cluster (non CRC) the routes hostnames must be added to `/etc/hosts`
in order to be accessible from a browser.

Below is the example of adding the route hostname to `/etc/hosts` using `hostctl`.

### Setup
1. [Install](https://guumaster.github.io/hostctl/docs/installation/) `hostctl`.

### Adding a host
After the central instance is provisioned and the routes are created, it's time to add the host to the `/etc/hosts`
#### Using the script

```
# requires: kubectl, hostctl, fzf, jq
./scripts/openshift-router.sh host add
```

#### Manually
```
# Get hostname
$ kubectl get routes managed-central-reencrypt -n rhacs-cblb0hq87d5sb2n3aesg -o jsonpath='{.spec.host}'
acs-cblb0hq87d5sb2n3aesg.kubernetes.docker.internal%

# Add host
$ sudo hostctl add domains acs acs-cblb0hq87d5sb2n3aesg.kubernetes.docker.internal

[✔] Domains 'acs-cblb0hq87d5sb2n3aesg.kubernetes.docker.internal' added.

+---------+--------+-----------+-----------------------------------------------------+
| PROFILE | STATUS |    IP     |                       DOMAIN                        |
+---------+--------+-----------+-----------------------------------------------------+
| acs     | on     | 127.0.0.1 | stage.foo.redhat.com                                |
| acs     | on     | 127.0.0.1 | acs-cblb0hq87d5sb2n3aesg.kubernetes.docker.internal |
+---------+--------+-----------+-----------------------------------------------------+
```
After the host is added, you can access it in a browser
### Removing the host
#### Using the script

```
# requires: kubectl, hostctl, fzf, jq
./scripts/openshift-router.sh host remove
```
#### Manually
When you are finished with testing the concrete instance, you can remove the host from the list
```
$ sudo hostctl remove domains acs acs-cblb0hq87d5sb2n3aesg.kubernetes.docker.internal
[✔] Domains 'acs-cblb0hq87d5sb2n3aesg.kubernetes.docker.internal' removed.

+---------+--------+-----------+----------------------+
| PROFILE | STATUS |    IP     |        DOMAIN        |
+---------+--------+-----------+----------------------+
| acs     | on     | 127.0.0.1 | stage.foo.redhat.com |
+---------+--------+-----------+----------------------+
```
