# fleetshard-sync

## Quickstart

```
# Start commands from git root directory

# Start fleet manager
$ ./scripts/setup-dev-env.sh

# Build and run fleetshard-sync
$ make fleetshard/build
$ OCM_TOKEN=$(ocm token) ./fleetshard-sync

# Create a central instace
$ ./scripts/create-central.sh
```
