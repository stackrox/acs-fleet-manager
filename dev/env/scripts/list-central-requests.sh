#!/usr/bin/env bash

# This script pretty-prints centrals in the database.
# JSON columns (routes, central, scanner) are decoded.
#
# Recommended settings for ~/.psqlrc:
#
#   \x on
#   \pset linestyle unicode
#   \pset border 2
#

set -e

query="select id, created_at, updated_at, deleted_at, cluster_id, name, namespace, status, encode(central, 'escape') as central, encode(scanner, 'escape') as scanner, encode(routes, 'escape') as routes from central_requests;"
psql -h localhost -U fleet_manager -d rhacsms -c "${query}"
