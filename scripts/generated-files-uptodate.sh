#!/usr/bin/env bash

set -e

echo "Verifying that generated files are up-to-date..."

make generate
git diff --exit-code HEAD
