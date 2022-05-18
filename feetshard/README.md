# fleetshard-sync

## Workflow

- Create a managed service instance
    - `curl -H "Authorization: Bearer ${OCM_TOKEN}" http://127.0.0.1:8000/api/dinosaurs_mgmt`

```
# Create a dinosaur
curl -X POST -H "Authorization: Bearer $(ocm token)" -H "Content-Type: application/json" http://127.0.0.1:8000/api/dinosaurs_mgmt/v1/dinosaurs\?async\=true -d '{"name": "test-dinosaur-1", "multi_az": true, "cloud_provider": "standalone", "region": "standalone"}'
curl -X GET -H "Authorization: Bearer $(ocm token)" -H "Content-Type: application/json" http://127.0.0.1:8000/api/dinosaurs_mgmt/v1/dinosaurs
```
