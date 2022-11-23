#!/bin/bash -e
# Temporary transitional script until https://gitlab.cee.redhat.com/service/app-interface/-/merge_requests/52332/diffs is merged.
exec "$(dirname "$0")/build_push_fleet_manager.sh" "$@"
