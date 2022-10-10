# Locust based load testing
Currently, it is an effortless setup for load testing. 
The purpose was to see if Fleet Manager has any suspicious resource consumption spikes.
Only one endpoint was stressed. The endpoint has a DB interaction under the hood.

You can find documented results of load testing on [app-interface Fleet Manager docs folder](https://gitlab.cee.redhat.com/service/app-interface/-/tree/master/docs/acs-fleet-manager/load-testing)

See locust doc for details: https://docs.locust.io/en/stable/quickstart.html

## Run load test with locust

1. Insert `INSERT_TOKEN` environment variable in `docker-compose,yml` file.
2. Start a master node and 10 workers using the following command:
```docker-compose up --scale worker=10```
3. Open Locust UI in browser http://localhost:8089/
4. Configure and run loadtest
