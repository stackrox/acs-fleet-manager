#!/usr/bin/env bash

die() {
    {
        # shellcheck disable=SC2059
        printf "$(date --iso-8601=ns) $*"
        echo
    } >&2
    exit 1
}

log() {
    # shellcheck disable=SC2059
    printf "$(date --iso-8601=ns) $*"
    echo
}
