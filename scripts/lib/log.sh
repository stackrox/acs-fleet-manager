#!/usr/bin/env bash

die() {
    {
        # shellcheck disable=SC2059
        printf "$*"
        echo
    } >&2
    exit 1
}

log() {
    # shellcheck disable=SC2059
    printf "$*"
    echo
}
