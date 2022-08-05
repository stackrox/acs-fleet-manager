# SOP : ACS Fleet Manager


- [SOP : ACS Fleet Manager](#sop--acs-fleet-manager)
    - [ACS Fleet Manager Down](#acs-fleet-manager-down)
        - [Impact](#impact)
        - [Summary](#summary)
        - [Access required](#access-required)
        - [Steps](#steps)
    - [ACS Fleet Manager availability](#acs-fleet-manager-availability)
        - [Impact](#impact-1)
        - [Summary](#summary-1)
        - [Access required](#access-required-1)
        - [Steps](#steps-1)
    - [ACS Fleet Manager latency](#acs-fleet-manager-latency)
        - [Impact](#impact-2)
        - [Summary](#summary-2)
        - [Access required](#access-required-2)
        - [Steps](#steps-2)
    - [ACS Central provisioning latency](#acs-central-provisioning-latency)
        - [Impact](#impact-3)
        - [Summary](#summary-3)
        - [Access required](#access-required-3)
        - [Steps](#steps-3)
    - [ACS Central provisioning correctness](#acs-central-provisioning-correctness)
        - [Impact](#impact-4)
        - [Summary](#summary-4)
        - [Access required](#access-required-4)
        - [Steps](#steps-4)
    - [ACS Central deletion correctness](#acs-central-deletion-correctness)
        - [Impact](#impact-5)
        - [Summary](#summary-5)
        - [Access required](#access-required-5)
        - [Steps](#steps-5)
    - [Escalations](#escalations)


---

## ACS Fleet Manager Down

### Impact

No incoming request can be received or processed.
The existing registered ACS Centrals will not be able to be processed.  
The ACS Centrals statuses will not be retrieved from OCM and updated to ACS Fleet Manager database.

### Summary

ACS Fleet Manager (all the replicas or pods) are down.

### Access required

- OSD console access to the cluster that runs the ACS Fleet Manager.
- Access to cluster resources: Pods/Deployments/Events

### Steps

- Check Deployments/acs-fleet-manager: check details page to make sure pods are configured and started; make sure pod number is configured to default: 3.
- Check cluster event logs to ensure there is no abnormality in the cluster level that could impact ACS Fleet Manager API.
    - Search Error/exception events with keywords "ACS Fleet Manager " and with text "image", "deployment" etc.
- Investigate the metrics in Grafana for any possible evidences of the crash.
    - Application: Volume, Latency, Error
        - Stage: https://grafana.stage.devshift.net/d/T2kek3H9a/acs-fleet-manager-slos?orgId=1&from=now-28d&to=now
        - Production: TODO
    - CPU, Network, Memory, IO
        - Stage: https://grafana.stage.devshift.net/d/T2kek3H9a/acs-fleet-manager-slos?orgId=1&from=now-28d&to=now
        - Production: TODO
- Check [openshift deployment template](https://github.com/stackrox/acs-fleet-manager/blob/main/templates/service-template.yml) for potential issue cause.
- Check [ACS Fleet Manager CI job logs](https://ci.ext.devshift.net/job/stackrox-acs-fleet-manager-gh-build-main/) for potential error cause.
- If necessary, escalate the incident to the corresponding teams.
    - Check [Escalations](#escalations) section below.

---

## ACS Fleet Manager availability

### Impact

Users are getting numerous amount of errors on API requests.

### Summary

ACS Fleet Manager is not performing normally and is returning an abnormally high number of 5xx Error requests.

### Access required

- OSD Console access to the cluster that runs the ACS Fleet Manager.
- Access to cluster resources: Pods/Deployments

### Steps

- Investigate the metrics in Grafana for any possible cause of the issue
    - Application: Volume, Latency, Error
        - Stage: https://grafana.stage.devshift.net/d/T2kek3H9a/acs-fleet-manager-slos?orgId=1&from=now-28d&to=now
        - Production: TODO
    - CPU, Network, Memory, IO
        - Stage: https://grafana.stage.devshift.net/d/T2kek3H9a/acs-fleet-manager-slos?orgId=1&from=now-28d&to=now
        - Production: TODO
- If there are container performance issue are identified (e.g.: CPU spike, high Latency etc.), increase the number of replicas.
- Check Deployments/acs-fleet-manager, check details page to make sure pods are configured and started. Start the pod if none is running (default:3).
- Check if the ACS Fleet Manager pods are running and verify the logs.
    ```
    #example
    oc get pods -n <acs-fleet-manager-stage|acs-fleet-manager-production>

    acs-fleet-manager-<pod_id>   1/1     Running
    acs-fleet-manager-<pod_id>   1/1     Running
    acs-fleet-manager-<pod_id>   1/1     Running

    # Check the pod logs to investigate possible causes of the issue (e.g. look for any Error/Exception messages)

    oc logs acs-fleet-manager-<pod_id>  | less
- If necessary, escalate the incident to the corresponding teams.
    - Check [Escalations](#escalations) section below.

---

## ACS Fleet Manager latency

### Impact

ACS Fleet Manager service is experiencing latency, or has been downgraded.

### Summary

ACS Fleet Manager is not performing normally and is not able to handle the load.

### Access required

- OSD Console access to the cluster that runs the ACS Fleet Manager.
- Access to cluster resources: Pods/Deployments

### Steps

refer to the steps in [ACS Fleet Manager availability](#acs-fleet-manager-availability)

---

## ACS Central provisioning latency

### Impact

ACS Fleet Manager service is experiencing issue while provisioning ACS centrals.

### Summary

ACS Fleet Manager is not able to perform acs central provisioning normally and is not able to handle the load.

### Access required

- OSD Console access to the cluster that runs the ACS Fleet Manager.
- - OSD Console access to the cluster that runs the Fleetshard-sync service.
- Access to cluster resources: Pods/Deployments/Events

### Steps

- Check if the ACS Fleet Manager pods are running and verify the logs.
    ```
    #example
    oc get pods -n <acs-fleet-manager-stage|acs-fleet-manager-production>

    acs-fleet-manager-<pod_id>   1/1     Running
    acs-fleet-manager-<pod_id>   1/1     Running
    acs-fleet-manager-<pod_id>   1/1     Running

    # Check the pod logs to investigate possible causes of the latency: look for Error/Exception message.

    oc logs acs-fleet-manager-<pod_id>  | less
    ```
  check the log to ensure Fleet Manager worker is started: there is exactly one Fleet Manager leader running.
    ```
    oc logs <pod-name> | grep 'Running as the leader.*FleetManager' 
     
    You should see output similar to the below from either one of the pods: 
    "Running as the leader and starting worker *workers.Worker"
    ```
- Check if the Fleetshard-sync services pods are running and verify the logs.
    ```
    #example
    oc get pods -n <fleetshard-sync-stage|fleetshard-sync-production>

    fleetshard-sync-<pod_id>   1/1     Running

    # Check the pod logs to investigate possible causes of the latency: look for Error/Exception message.

    oc logs fleetshard-sync-<pod_id>  | less
    ```
  check the log to ensure Fleetshard-sync is started and reconcile loops start for requested centrals.
    ```
    oc logs <pod-name> | grep 'Start reconcile central' 
     
    You should see output similar to the below: 
    "Start reconcile central <central_name>"
    ```
- How to handle:
    - Error/exception appears related to Fleet Manager API or no leader worker is running, try to restart the pods.
    - Error/exception appears related to Fleetshard-sync check with ACS team Data Plane support.
    - Otherwise, or if unsure about the reason, escalate the issue to the Control Plane team.

---

## ACS Central provisioning correctness

### Impact

ACS Fleet Manager service is experiencing issue while provisioning ACS centrals.

### Summary

ACS Fleet Manager is not able to provision ACS centrals correctly.

### Access required

- OSD Console access to the cluster that runs the ACS Fleet Manager.
- OSD Console access to the cluster that runs the Fleetshard-sync service.
- Access to cluster resources: Pods/Deployments

### Steps

refer to the steps [ACS Central provisioning latency](#acs-central-provisioning-latency)

---

## ACS Central deletion correctness

### Impact

ACS Fleet Manager service is experiencing issue while deleting ACS centrals.

### Summary

ACS Fleet Manager is not able to performing ACS central deletion correctly.

### Access required

- OSD Console access to the cluster that runs the ACS Fleet Manager.
- OSD Console access to the cluster that runs the Fleetshard-sync service.
- Access to cluster resources: Pods/Deployments

### Steps

refer to the steps [ACS Central provisioning latency](#acs-central-provisioning-latency)
 
---

## Escalations

- Error/exception appears related to ACS Fleet Manager API or no leader worker is running, try to restart the pods.
- Error/exception related to OCM, check with OCM support to see if they've received OSD cluster request.
- Error/exception events found in the OSD cluster level, check with OCM support.
- Error/exception related to SSO outage, check with CIAM team.
- Error/exception related to fleetshard-sync, check with ACS team or Data Plane support.
- Otherwise, or if unsure about the reason, escalate the issue to the Control Plane team 
