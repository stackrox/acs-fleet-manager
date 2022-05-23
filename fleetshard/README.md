# fleetshard-sync

## Workflow

- Create a managed service instance
    - `curl -H "Authorization: Bearer ${OCM_TOKEN}" http://127.0.0.1:8000/api/rhacs/`

```
# Create a dinosaur
curl -X POST -H "Authorization: Bearer $(ocm token)" -H "Content-Type: application/json" http://127.0.0.1:8000/api/rhacs/v1/centrals?async\=true -d '{"name": "test-rhacs-1", "multi_az": true, "cloud_provider": "standalone", "region": "standalone"}'
curl -X GET -H "Authorization: Bearer $(ocm token)" -H "Content-Type: application/json" http://127.0.0.1:8000/api/rhacs/v1/centrals/v1/dinosaurs
```

## Start fleet-manager

```
# Create OCM token <link to docs>
# export OCM_TOKEN=$(ocm token)

# docker rm fleet-manager-db
make secrets/touch
make db/setup && make db/migrate
make binary
./fleet-manager serve
```

