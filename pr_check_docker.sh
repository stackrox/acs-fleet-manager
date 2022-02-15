#!/bin/bash -ex

# reflects defaults found in pkg/config/db.go
export GORM_DIALECT="postgres"
export GORM_HOST="localhost"
export GORM_PORT="5432"
export GORM_NAME="managed-services-api"
export GORM_USERNAME="managed-services-api"
export GORM_PASSWORD="secret"
export GORM_SSLMODE="disable"
export GORM_DEBUG="false"

export LOGLEVEL="1"
export TEST_SUMMARY_FORMAT="standard-verbose"

# cd /fleet-manager/src/github.com/bf2fc6cc711aee1a0c2a/fleet-manager
# ls -la 
# go version

# start postgres
which pg_ctl
PGDATA=/var/lib/postgresql/data /usr/lib/postgresql/*/bin/pg_ctl -w stop
PGDATA=/var/lib/postgresql/data /usr/lib/postgresql/*/bin/pg_ctl start -o "-c listen_addresses='*' -p 5432"

# check the code. Then run the unit and integration tests and cleanup cluster (if running against real OCM)
make -k lint verify test test/integration test/cluster/cleanup

# required for entrypoint script run by docker to exit and stop container
exit 0