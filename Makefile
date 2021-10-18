PACKAGES=$(shell go list ./... | grep -v '/simulation')
COMMIT := $(shell git log -1 --format='%H')
DOCKER := $(shell which docker)
DOCKER_BUF := $(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace bufbuild/buf


proto-gen:
	$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace tendermintdev/sdk-proto-gen sh ./scripts/protocgen.sh

proto-lint:
	@$(DOCKER_BUF) lint --error-format=json

test: test-unit test-build

test-all: check test-race test-cover

test-unit:
	@VERSION=$(VERSION) go test -mod=readonly ./...

test-race:
	@VERSION=$(VERSION) go test -mod=readonly -race ./...

proto-check-breaking:
	@$(DOCKER_BUF) breaking --against .git#branch=master
.PHONY: proto-check-breaking

proto-check-breaking-ci:
	@$(DOCKER_BUF) breaking --against $(HTTPS_GIT)#branch=master
.PHONY: proto-check-breaking-ci