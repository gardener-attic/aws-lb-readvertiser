# Copyright (c) 2017 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

IMAGE_REPOSITORY := eu.gcr.io/gardener-project/gardener/aws-lb-readvertiser
IMAGE_TAG        := $(shell cat VERSION)

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: revendor
revendor:
	@dep ensure -update

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
	@.ci/check
