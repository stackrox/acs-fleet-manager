#!/usr/bin/env bash

die() {
    log "$@" >&2
    exit 1
}

log() {
    # shellcheck disable=SC2059
    printf "$(date "+%Y-%m-%dT%H:%M:%S,%N%:z") $*"
    echo
}
