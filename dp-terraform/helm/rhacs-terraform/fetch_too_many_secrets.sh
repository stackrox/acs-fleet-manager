#/bin/bash

set -o errexit   # -e
set -o pipefail
set -o nounset   # -u

export OUTPUT_FILE="/var/tmp/tmp_secrets/secrets.yaml"
export OUTPUT_DIRECTORY="$(dirname ${OUTPUT_FILE})"

# Requires BitWarden CLI (for now)
# To install: sudo snap install bw

# If you're testing this script repeatedly, it's worth logging in and storing
# the session key *before* running it, which will keep the session key across
# runs. Otherwise the export here will only last for the script's duration.

# Check if we need to get a new BitWarden CLI Session Key.
if [[ -z "$BW_SESSION" ]]; then
    if bw login --check; then
        # We don't have a session key but we are logged in, so unlock and store the session.
        export BW_SESSION=$(bw unlock --raw)
    else
        # We don't have a session key and are not logged in, so log in and store the session.
        export BW_SESSION=$(bw login --raw)
    fi
fi

# Create a directory to store private temporary secret files
umask 077  # Disable Group and Other rwx
mkdir -p "${OUTPUT_DIRECTORY}"
chmod 1700 "${OUTPUT_DIRECTORY}"

# TODO: Write helpers to extract specific fields, including by bash variable as field name.

FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_ID="rhacs-fleetshard-staging"
# TODO: Handle multiple environments - right now this assumes staging!
# Unlike Observability tokens that exist in a single bitwarden item, the fleetshard
# sync red hat sso client secret is one item per environment. This means that we would
# need something fancier to handle environment selection. For now, we assume staging.
FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_SECRET=$(bw get password 028ce1a9-f751-4056-9c72-aea70052728b)
LOGGING_AWS_ACCESS_KEY_ID=$(bw get item "84e2d673-27dd-4e87-bb16-aee800da4d73" | jq '.fields[] | select(.name | contains("AccessKeyID")) | .value' --raw-output)
LOGGING_AWS_SECRET_ACCESS_KEY=$(bw get item "84e2d673-27dd-4e87-bb16-aee800da4d73" | jq '.fields[] | select(.name | contains("SecretAccessKey")) | .value' --raw-output)
OBSERVABILITY_GITHUB_ACCESS_TOKEN=$(bw get password eb7aecd3-b553-4999-b201-aebe01445822)
OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID="observatorium-rhacs-metrics-staging"
OBSERVABILITY_OBSERVATORIUM_METRICS_SECRET=$(
    bw get item 510c8ed9-ba9f-46d9-b906-ae6100cf72f5 | \
    jq --arg OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID "${OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID}" \
        '.fields[] | select(.name | contains($OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID)) | .value' --raw-output
)

cat <<EOF > ${OUTPUT_FILE}
fleetshardSync:
  redHatSSO:
    clientId: ${FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_ID}
    clientSecret: ${FLEETSHARD_SYNC_RED_HAT_SSO_CLIENT_SECRET}
logging:
  aws:
    accessKeyId: ${LOGGING_AWS_ACCESS_KEY_ID}
    secretAccessKey: ${LOGGING_AWS_SECRET_ACCESS_KEY}
observability:
  github:
    accessToken: ${OBSERVABILITY_GITHUB_ACCESS_TOKEN}
  observatorium:
    metricsClientId: ${OBSERVABILITY_OBSERVATORIUM_METRICS_CLIENT_ID}
    metricsSecret: ${OBSERVABILITY_OBSERVATORIUM_METRICS_SECRET}
EOF

echo "Secrets successfully written to ${OUTPUT_FILE}"