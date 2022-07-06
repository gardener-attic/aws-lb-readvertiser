#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -e

GOLANGCI_LINT_CONFIG_FILE=""

for arg in "$@"; do
  case $arg in
    --golangci-lint-config=*)
    GOLANGCI_LINT_CONFIG_FILE="--config ${arg#*=}"
    shift
    ;;
  esac
done

# Execute lint checks.
golangci-lint run ${GOLANGCI_LINT_CONFIG_FILE} --timeout 10m "$@"

# Run tests
go test ./...