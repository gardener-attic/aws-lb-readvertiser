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
VERSION          := $(shell cat VERSION)
LD_FLAGS         := "-w -X $(REPOSITORY)/pkg/version.Version=$(VERSION)"
PACKAGES         := $(shell go list ./... | grep -v '/vendor/')
REGISTRY         := eu.gcr.io/sap-cloud-platform-dev1
IMAGE_REPOSITORY := $(REGISTRY)/garden/$(PROJECT)
IMAGE_TAG        := $(VERSION)

BIN_DIR          := bin
GOBIN            := $(PWD)/bin
PATH             := $(GOBIN):$(PATH)
USER             :=  $(shell id -u -n)

export PATH
export GOBIN

.PHONY: verify
verify: vet fmt lint test

.PHONY: revendor
revendor:
	@dep ensure -update
	@dep prune


.PHONY: build
build:
	@go build -o $(BIN_DIR)/aws-lb-readvertiser $(GO_EXTRA_FLAGS) -ldflags $(LD_FLAGS) main.go

.PHONY: release
release: build docker-build docker-image docker-login docker-push rename-binaries

.PHONY: build-release
build-release:
	@go build -o /go/bin/aws-lb-readvertiser $(GO_EXTRA_FLAGS) -ldflags $(LD_FLAGS) main.go

.PHONY: docker-build
docker-build:
	@./hack/build-release
	@sudo chown $(user):$(group) $(BIN_DIR)/rel/aws-lb-readvertiser

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
	@if [[ -f $(BIN_DIR)/aws-lb-readvertiser ]]; then cp $(BIN_DIR)/aws-lb-readvertiser aws-lb-readvertiser-darwin-amd64; fi
	@if [[ -f $(BIN_DIR)/rel/aws-lb-readvertiser ]]; then cp $(BIN_DIR)/rel/aws-lb-readvertiser aws-lb-readvertiser-linux-amd64; fi


.PHONY: clean
clean:
	@rm -rf $(BIN_DIR)/
	@rm -f *linux-amd64
	@rm -f *darwin-amd64

.PHONY: fmt
fmt:
	@go fmt $(PACKAGES)

.PHONY: vet
vet:
	@go vet $(PACKAGES)

.PHONY: lint
lint:
	@for package in $(PACKAGES); do \
		golint -set_exit_status $$(find $$package -maxdepth 1 -name "*.go" | grep -vE 'zz_generated|_test.go') || exit 1; \
	done

.PHONY: test
test:
	@ginkgo -r pkg
