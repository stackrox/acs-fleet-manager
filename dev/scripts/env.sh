#!/usr/bin/env bash
set -eo pipefail

# As we change the pwd, this must be a function and can't be a standalone
# script.
function cdffm() {
	[[ -n "$GOENV_GOPATH" ]] || { echo >&2 "GOPATH could not be determined"; return 1; }
	# if an arg is provided, attempt to cd into that directory,
	# defaulting to stackrox.
	repo="${1:-acs-fleet-manager}"
	cd "${GOENV_GOPATH}/src/github.com/stackrox/${repo}"
}

function ffm_curl {

}
