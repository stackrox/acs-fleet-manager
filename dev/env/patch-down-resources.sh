#!/bin/sh
kubectl -n test-central-1 patch deploy/central --patch '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"limits":{"cpu":"1","memory":"4Gi"},"requests":{"cpu":"1","memory":"1Gi"}}}]}}}}'
