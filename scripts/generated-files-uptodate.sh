#!/usr/bin/env bash

set -e

echo "Verifying that generated files are up-to-date..."

for f in "$@"; do
    echo "File ${f} has been modified"
done

make generate
git diff --exit-code HEAD
