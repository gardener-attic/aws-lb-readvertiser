# Copyright 2017 The Gardener Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

VCS              := github.com
ORGANIZATION     := gardener
PROJECT          := aws-lb-readvertiser
REPOSITORY       := $(VCS)/$(ORGANIZATION)/$(PROJECT)
VERSION          := v0.0.2
LD_FLAGS         := "-w -X $(REPOSITORY)/pkg/version.Version=$(VERSION)"
PACKAGES         := $(shell go list ./... | grep -v '/vendor/')
REGISTRY         := eu.gcr.io/sap-cloud-platform-dev1
IMAGE_REPOSITORY := $(REGISTRY)/garden/$(PROJECT)
IMAGE_TAG        := $(VERSION)

GOBIN            := $(PWD)/bin
PATH             := $(GOBIN):$(PATH)
USER             :=  $(shell id -u -n)

export PATH
export GOBIN

.PHONY: verify
verify: vet fmt lint test

.PHONY: revendor
revendor:
	@glide up -v
	@glide-vc --use-lock-file --no-tests --only-code

.PHONY: build
build: bin/aws-lb-readvertiser

.PHONY: release
release: build docker-build docker-image docker-login docker-push rename-binaries clean

bin/aws-lb-readvertiser: check-go-version
	@go install -v -ldflags $(LD_FLAGS) $(REPOSITORY)/

.PHONY: build-release
build-release:
	@go build -o /go/bin/aws-lb-readvertiser -v -ldflags $(LD_FLAGS) $(REPOSITORY)/

.PHONY: docker-build
docker-build: rel/bin/aws-lb-readvertiser

rel/bin/aws-lb-readvertiser:
	@./scripts/build-release
	@sudo chown $(user):$(group) rel/bin/aws-lb-readvertiser

.PHONY: docker-image
docker-image:
	@if [[ ! -f rel/bin/aws-lb-readvertiser ]]; then echo "No binary found. Please run 'make docker-build'"; false; fi
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
	@cp bin/aws-lb-readvertiser aws-lb-readvertiser-darwin-amd64
	@cp rel/bin/aws-lb-readvertiser aws-lb-readvertiser-linux-amd64

.PHONY: fmt
fmt:
	@go fmt $(PACKAGES)

.PHONY: vet
vet:
	@go vet $(PACKAGES)

.PHONY: lint
lint:
	@for package in $(PACKAGES); do \
		golint -set_exit_status $$package $$i || exit 1; \
	done

.PHONY: test
test:
	@go test $(PACKAGES)

.PHONY: check-go-version
check-go-version:
	@./scripts/check-go-version

.PHONY: clean
clean:
	@rm -rf bin/
	@rm -rf rel/
