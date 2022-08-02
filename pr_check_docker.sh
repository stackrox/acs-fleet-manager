#!/bin/bash -ex

export LOGLEVEL="1"
export TEST_SUMMARY_FORMAT="standard-verbose"

# cd /fleet-manager/src/github.com/stackrox/acs-fleet-manager
# ls -la
# go version

# start postgres
which pg_ctl
# shellcheck disable=SC2037,SC2211
PGDATA=/var/lib/postgresql/data /usr/lib/postgresql/*/bin/pg_ctl -w stop
# shellcheck disable=SC2037,SC2211
PGDATA=/var/lib/postgresql/data /usr/lib/postgresql/*/bin/pg_ctl start -o "-c listen_addresses='*' -p 5432"

# check the code. Then run the unit and integration tests and cleanup cluster (if running against real OCM)
make -k lint verify test test/integration test/cluster/cleanup

# required for entrypoint script run by docker to exit and stop container
exit 0
