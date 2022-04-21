# fleetshard-sync

## Workflow

 - Create a managed service instance
   - `curl -H "Authorization: Bearer ${OCM_TOKEN}" http://127.0.0.1:8000/api/dinosaurs_mgmt`

```
# Create a dinosaur
curl -X POST -H "Authorization: Bearer $(ocm token)" http://127.0.0.1:8000/api/dinosaurs_mgmt/v1/dinosaurs\?async\=true -d ' {"name": "dev-acs-instance", "multi_az": true}'
```
