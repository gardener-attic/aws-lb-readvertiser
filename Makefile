# SPDX-FileCopyrightText: 2017 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

IMAGE_REPOSITORY              := europe-docker.pkg.dev/gardener-project/public/gardener/aws-lb-readvertiser
IMAGE_TAG                     := $(shell cat VERSION)
GOLANGCI_LINT_CONFIG_FILE     := "./.golangci.yaml"
REPO_ROOT                     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: install-requirements
install-requirements:
	# install golangci-lint
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2

.PHONY: revendor
revendor:
	@go mod tidy
	@go mod vendor

.PHONY: build
build:
	@.ci/build

.PHONY: build-local
build-local:
	@env LOCAL_BUILD=1 .ci/build

.PHONY: release
release: build build-local docker-image docker-login docker-push rename-binaries

.PHONY: docker-image
docker-image:
	@if [[ ! -f bin/rel/aws-lb-readvertiser ]]; then echo "No binary found. Please run 'make build'"; false; fi
	@docker build -t $(IMAGE_REPOSITORY):$(IMAGE_TAG) --rm .

.PHONY: docker-login
docker-login:
	@gcloud auth activate-service-account --key-file .kube-secrets/gcr/gcr-readwrite.json

.PHONY: docker-push
docker-push:
	@if ! docker images $(IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-image'"; false; fi
	@gcloud docker -- push $(IMAGE_REPOSITORY):$(IMAGE_TAG)

.PHONY: rename-binaries
rename-binaries:
	@if [[ -f bin/aws-lb-readvertiser ]]; then cp bin/aws-lb-readvertiser aws-lb-readvertiser-darwin-amd64; fi
	@if [[ -f bin/rel/aws-lb-readvertiser ]]; then cp bin/rel/aws-lb-readvertiser aws-lb-readvertiser-linux-amd64; fi

.PHONY: clean
clean:
	@rm -rf bin/
	@rm -f *linux-amd64
	@rm -f *darwin-amd64

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: verify
verify: check

.PHONY: check
check:
	@$(REPO_ROOT)/hack/check.sh --golangci-lint-config=$(GOLANGCI_LINT_CONFIG_FILE) . ./controller/...
