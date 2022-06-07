# How to use

This directory defines a docker compose based test environment which is easy to spin up.

Make sure that no other DB is running:
```
$ make -C $(git rev-parse --show-toplevel) db/teardown
```

Either source `dev/env/.envrc` or use `direnv` for loading this environment automatically.
This will make sure that some settings are available in the current environment, which are
then picked up by `docker-compose.yaml`. In particular, this includes OCM tokens and secrets
from `secrets/`. If the OCM token needs to be refreshed, re-source `dev/env/.envrc` or use
`direnv reload`.

A Docker daemon needs to be running and the `docker` CLI needs to be installed.

Spin up the test environment using:
```
$ docker compose up
```

The `fleet-manager` service will be reachable at `localhost:8000`.

If new Docker images for `fleet-manager` and `fleetshard-sync` shall be built, use
`docker compose up --build`.

To shut down the test environment:
```
$ docker compose down
```

If the Postgres volume shall also be deleted:
```
$ docker compose down -v
```
