# Copyright 2018 The Trickster Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

-include ci.mk release.mk

DEFAULT: build

PROJECT_DIR    := $(shell pwd)
GO             ?= go
GOFMT          ?= $(GO)fmt
FIRST_GOPATH   := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
TRICKSTER_MAIN := cmd/trickster
TRICKSTER      := $(FIRST_GOPATH)/bin/trickster
BUILD_TIME     := $(shell date -u +%FT%T%z)
GIT_LATEST_COMMIT_ID     ?= $(shell git rev-parse HEAD)
IMAGE_TAG      ?= latest
IMAGE_ARCH     ?= $(shell $(GO) env GOARCH)
GOARCH         ?= $(shell $(GO) env GOARCH)
TAGVER         ?= $(shell git describe --tags --dirty --always)
LDFLAGS         =-ldflags "-extldflags '-static' -w -s -X main.applicationBuildTime=$(BUILD_TIME) -X main.applicationGitCommitID=$(GIT_LATEST_COMMIT_ID) -X main.applicationVersion=$(TAGVER)"
BUILD_SUBDIR   := bin
PACKAGE_DIR    := ./$(BUILD_SUBDIR)/trickster-$(TAGVER)
BIN_DIR        := $(PACKAGE_DIR)/bin
CONF_DIR       := $(PACKAGE_DIR)/conf
CGO_ENABLED    ?= 0
BUMPER_FILE    := ./testdata/license_header_template.txt

.PHONY: go-mod-vendor
go-mod-vendor:
	$(GO) mod vendor

.PHONY: go-mod-tidy
go-mod-tidy:
	$(GO) mod tidy

.PHONY: test-go-mod
test-go-mod:
	@git diff --quiet --exit-code go.mod go.sum || echo "There are changes to go.mod and go.sum which needs to be committed"

BUILD_FLAGS ?= -a -v
.PHONY: build
build: go-mod-tidy go-mod-vendor
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(LDFLAGS) $(BUILD_FLAGS) -o ./$(BUILD_SUBDIR)/trickster  $(TRICKSTER_MAIN)/*.go

<<<<<<< HEAD
rpm: build
	mkdir -p ./$(BUILD_SUBDIR)/SOURCES
	cp -p ./$(BUILD_SUBDIR)/trickster ./$(BUILD_SUBDIR)/SOURCES/
	cp deploy/systemd/trickster.service ./$(BUILD_SUBDIR)/SOURCES/
	sed -e 's%^# log_file:.*$$%log_file: /var/log/trickster/trickster.log%' \
		-e 's%prometheus:9090%localhost:9090%' \
		< examples/conf/example.full.yaml > ./$(BUILD_SUBDIR)/SOURCES/trickster.yaml
	rpmbuild --define "_topdir $(CURDIR)/$(BUILD_SUBDIR)" \
		--define "_version $(TAGVER)" \
		--define "_release 1" \
		-ba deploy/packaging/trickster.spec
=======
SRC_PKGS := cmd pkg # directories which hold app source excluding tests (not vendored)
SRC_DIRS := $(SRC_PKGS) # directories which hold app source (not vendored)

DOCKER_PLATFORMS := linux/amd64 linux/arm64
BIN_PLATFORMS    := $(DOCKER_PLATFORMS)

# Used internally.  Users should pass GOOS and/or GOARCH.
OS   := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
ARCH := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))

BASEIMAGE_PROD   ?= alpine
BASEIMAGE_DBG    ?= debian:12
BASEIMAGE_UBI    ?= registry.access.redhat.com/ubi10/ubi-minimal

IMAGE            := $(REGISTRY)/$(BIN)
VERSION_PROD     := $(VERSION)
VERSION_DBG      := $(VERSION)-dbg
VERSION_UBI      := $(VERSION)-ubi
TAG              := $(VERSION)_$(OS)_$(ARCH)
TAG_PROD         := $(TAG)
TAG_DBG          := $(VERSION)-dbg_$(OS)_$(ARCH)
TAG_UBI          := $(VERSION)-ubi_$(OS)_$(ARCH)

GO_VERSION       ?= 1.24
BUILD_IMAGE      ?= ghcr.io/appscode/golang-dev:$(GO_VERSION)
CHART_TEST_IMAGE ?= quay.io/helmpack/chart-testing:v3.13.0

OUTBIN = bin/$(BIN)-$(OS)-$(ARCH)
ifeq ($(OS),windows)
  OUTBIN := bin/$(BIN)-$(OS)-$(ARCH).exe
  BIN := $(BIN).exe
endif

# Directories that we need created to build/test.
BUILD_DIRS  := bin/$(OS)_$(ARCH)     \
               .go/bin/$(OS)_$(ARCH) \
               .go/cache             \
               hack/config           \
               $(HOME)/.credentials  \
               $(HOME)/.kube         \
               $(HOME)/.minikube

DOCKERFILE_PROD  = Dockerfile.in
DOCKERFILE_DBG   = Dockerfile.dbg
DOCKERFILE_UBI   = Dockerfile.ubi

DOCKER_REPO_ROOT := /go/src/$(GO_PKG)/$(REPO)

# If you want to build all binaries, see the 'all-build' rule.
# If you want to build all containers, see the 'all-container' rule.
# If you want to build AND push all containers, see the 'all-push' rule.
all: fmt build

# For the following OS/ARCH expansions, we transform OS/ARCH into OS_ARCH
# because make pattern rules don't match with embedded '/' characters.

build-%:
	@$(MAKE) build                        \
	    --no-print-directory              \
	    GOOS=$(firstword $(subst _, ,$*)) \
	    GOARCH=$(lastword $(subst _, ,$*))

container-%:
	@$(MAKE) container                    \
	    --no-print-directory              \
	    GOOS=$(firstword $(subst _, ,$*)) \
	    GOARCH=$(lastword $(subst _, ,$*))

push-%:
	@$(MAKE) push                         \
	    --no-print-directory              \
	    GOOS=$(firstword $(subst _, ,$*)) \
	    GOARCH=$(lastword $(subst _, ,$*))

all-build: $(addprefix build-, $(subst /,_, $(BIN_PLATFORMS)))
ifeq ($(COMPRESS),yes)
	@cd bin; \
	sha256sum $(patsubst $(BIN)-windows-%.tar.gz,$(BIN)-windows-%.zip, $(addsuffix .tar.gz, $(addprefix $(BIN)-, $(subst /,-, $(BIN_PLATFORMS))))) > $(BIN)-checksums.txt
endif

all-container: $(addprefix container-, $(subst /,_, $(DOCKER_PLATFORMS)))

all-push: $(addprefix push-, $(subst /,_, $(DOCKER_PLATFORMS)))

version:
	@echo ::set-output name=version::$(VERSION)
	@echo ::set-output name=version_strategy::$(version_strategy)
	@echo ::set-output name=git_tag::$(git_tag)
	@echo ::set-output name=git_branch::$(git_branch)
	@echo ::set-output name=commit_hash::$(commit_hash)
	@echo ::set-output name=commit_timestamp::$(commit_timestamp)

.PHONY: gen
gen:
	@true

fmt: $(BUILD_DIRS)
	@docker run                                                 \
	    -i                                                      \
	    --rm                                                    \
	    -u $$(id -u):$$(id -g)                                  \
	    -v $$(pwd):/src                                         \
	    -w /src                                                 \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin                \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin/$(OS)_$(ARCH)  \
	    -v $$(pwd)/.go/cache:/.cache                            \
	    --env HTTP_PROXY=$(HTTP_PROXY)                          \
	    --env HTTPS_PROXY=$(HTTPS_PROXY)                        \
	    $(BUILD_IMAGE)                                          \
	    /bin/bash -c "                                          \
	        REPO_PKG=$(GO_PKG)                                  \
	        ./hack/fmt.sh $(SRC_DIRS)                           \
	    "

build: $(OUTBIN)

# The following structure defeats Go's (intentional) behavior to always touch
# result files, even if they have not changed.  This will still run `go` but
# will not trigger further work if nothing has actually changed.

$(OUTBIN): .go/$(OUTBIN).stamp
	@true

# This will build the binary under ./.go and update the real binary iff needed.
.PHONY: .go/$(OUTBIN).stamp
.go/$(OUTBIN).stamp: $(BUILD_DIRS)
	@echo "making $(OUTBIN)"
	@docker run                                                 \
	    -i                                                      \
	    --rm                                                    \
	    -u $$(id -u):$$(id -g)                                  \
	    -v $$(pwd):/src                                         \
	    -w /src                                                 \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin                \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin/$(OS)_$(ARCH)  \
	    -v $$(pwd)/.go/cache:/.cache                            \
	    --env HTTP_PROXY=$(HTTP_PROXY)                          \
	    --env HTTPS_PROXY=$(HTTPS_PROXY)                        \
	    $(BUILD_IMAGE)                                          \
	    /bin/bash -c "                                          \
	        PRODUCT_OWNER_NAME=$(PRODUCT_OWNER_NAME)            \
	        PRODUCT_NAME=$(PRODUCT_NAME)                        \
	        ENFORCE_LICENSE=$(ENFORCE_LICENSE)                  \
	        ARCH=$(ARCH)                                        \
	        OS=$(OS)                                            \
	        VERSION=$(VERSION)                                  \
	        version_strategy=$(version_strategy)                \
	        git_branch=$(git_branch)                            \
	        git_tag=$(git_tag)                                  \
	        commit_hash=$(commit_hash)                          \
	        commit_timestamp=$(commit_timestamp)                \
	        ./hack/build.sh                                     \
	    "
	@if ! cmp -s .go/bin/$(OS)_$(ARCH)/$(BIN) $(OUTBIN); then   \
	    mv .go/bin/$(OS)_$(ARCH)/$(BIN) $(OUTBIN);              \
	    date >$@;                                               \
	fi
ifeq ($(COMPRESS),yes)
ifeq ($(OS),windows)
	@echo "compressing $(OUTBIN)";                               \
	cd bin;                                                      \
	zip -j $(subst .exe,,$(BIN))-$(OS)-$(ARCH).zip $(subst .exe,,$(BIN))-$(OS)-$(ARCH).exe ../LICENSE
else
	@echo "compressing $(OUTBIN)";                               \
	cd bin;                                                      \
	tar -czvf $(BIN)-$(OS)-$(ARCH).tar.gz $(BIN)-$(OS)-$(ARCH) ../LICENSE
endif
endif
	@echo

# Used to track state in hidden files.
DOTFILE_IMAGE    = $(subst /,_,$(IMAGE))-$(TAG)

container: bin/.container-$(DOTFILE_IMAGE)-PROD bin/.container-$(DOTFILE_IMAGE)-DBG bin/.container-$(DOTFILE_IMAGE)-UBI
ifeq (,$(SRC_REG))
bin/.container-$(DOTFILE_IMAGE)-%: bin/$(BIN)-$(OS)-$(ARCH) $(DOCKERFILE_%)
	@echo "container: $(IMAGE):$(TAG_$*)"
	@sed                                  \
		-e 's|{ARG_BIN}|$(BIN)|g'           \
		-e 's|{ARG_ARCH}|$(ARCH)|g'         \
		-e 's|{ARG_OS}|$(OS)|g'             \
		-e 's|{ARG_FROM}|$(BASEIMAGE_$*)|g' \
		-e 's|{ARG_TAG}|$(TAG)|g'           \
		$(DOCKERFILE_$*) > bin/.dockerfile-$*-$(OS)_$(ARCH)
	@docker buildx build --platform $(OS)/$(ARCH) --load --pull -t $(IMAGE):$(TAG_$*) -f bin/.dockerfile-$*-$(OS)_$(ARCH) .
	@docker images -q $(IMAGE):$(TAG_$*) > $@
	@echo
else
bin/.container-$(DOTFILE_IMAGE)-%:
	@echo "container: $(IMAGE):$(TAG_$*)"
	@docker tag $(SRC_REG)/$(BIN):$(TAG_$*) $(IMAGE):$(TAG_$*)
	@echo
endif

push: bin/.push-$(DOTFILE_IMAGE)-PROD bin/.push-$(DOTFILE_IMAGE)-DBG bin/.push-$(DOTFILE_IMAGE)-UBI
bin/.push-$(DOTFILE_IMAGE)-%: bin/.container-$(DOTFILE_IMAGE)-%
	@docker push $(IMAGE):$(TAG_$*)
	@echo "pushed: $(IMAGE):$(TAG_$*)"
	@echo

.PHONY: docker-manifest
docker-manifest: docker-manifest-PROD docker-manifest-DBG docker-manifest-UBI
docker-manifest-%:
	@docker manifest create -a $(IMAGE):$(VERSION_$*) $(foreach PLATFORM,$(DOCKER_PLATFORMS),$(IMAGE):$(VERSION_$*)_$(subst /,_,$(PLATFORM)))
	@docker manifest push $(IMAGE):$(VERSION_$*)

.PHONY: docker-certify-redhat
docker-certify-redhat:
	@preflight check container $(IMAGE):$(VERSION_UBI) \
		--submit \
		--certification-component-id=69423549c3532f69bf47190d

.PHONY: test
test: unit-tests e2e-tests

unit-tests: $(BUILD_DIRS)
	@docker run                                                 \
	    -i                                                      \
	    --rm                                                    \
	    -u $$(id -u):$$(id -g)                                  \
	    -v $$(pwd):/src                                         \
	    -w /src                                                 \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin                \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin/$(OS)_$(ARCH)  \
	    -v $$(pwd)/.go/cache:/.cache                            \
	    --env HTTP_PROXY=$(HTTP_PROXY)                          \
	    --env HTTPS_PROXY=$(HTTPS_PROXY)                        \
	    $(BUILD_IMAGE)                                          \
	    /bin/bash -c "                                          \
	        ARCH=$(ARCH)                                        \
	        OS=$(OS)                                            \
	        VERSION=$(VERSION)                                  \
	        ./hack/test.sh $(SRC_PKGS)                          \
	    "

# - e2e-tests can hold both ginkgo args (as GINKGO_ARGS) and program/test args (as TEST_ARGS).
#       make e2e-tests TEST_ARGS="--selfhosted-operator=false --storageclass=standard" GINKGO_ARGS="--flakeAttempts=2"
#
# - Minimalist:
#       make e2e-tests
#
# NB: -t is used to catch ctrl-c interrupt from keyboard and -t will be problematic for CI.

GINKGO_ARGS ?=
TEST_ARGS   ?=

.PHONY: e2e-tests
e2e-tests: $(BUILD_DIRS)
	@docker run                                                 \
	    -i                                                      \
	    --rm                                                    \
	    -u $$(id -u):$$(id -g)                                  \
	    -v $$(pwd):/src                                         \
	    -w /src                                                 \
	    --net=host                                              \
	    -v $(HOME)/.kube:/.kube                                 \
	    -v $(HOME)/.minikube:$(HOME)/.minikube                  \
	    -v $(HOME)/.credentials:$(HOME)/.credentials            \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin                \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin/$(OS)_$(ARCH)  \
	    -v $$(pwd)/.go/cache:/.cache                            \
	    --env HTTP_PROXY=$(HTTP_PROXY)                          \
	    --env HTTPS_PROXY=$(HTTPS_PROXY)                        \
	    --env KUBECONFIG=$(KUBECONFIG)                          \
	    --env-file=$$(pwd)/hack/config/.env                     \
	    $(BUILD_IMAGE)                                          \
	    /bin/bash -c "                                          \
	        ARCH=$(ARCH)                                        \
	        OS=$(OS)                                            \
	        VERSION=$(VERSION)                                  \
	        DOCKER_REGISTRY=$(REGISTRY)                         \
	        TAG=$(TAG)                                          \
	        KUBECONFIG=$${KUBECONFIG#$(HOME)}                   \
	        GINKGO_ARGS='$(GINKGO_ARGS)'                        \
	        TEST_ARGS='$(TEST_ARGS)'                            \
	        ./hack/e2e.sh                                       \
	    "

.PHONY: e2e-parallel
e2e-parallel:
	@$(MAKE) e2e-tests GINKGO_ARGS="-p -stream --flakeAttempts=2" --no-print-directory

ADDTL_LINTERS   := goconst,gofmt,goimports,unparam

.PHONY: lint
lint: $(BUILD_DIRS)
	@echo "running linter"
	@docker run                                                 \
	    -i                                                      \
	    --rm                                                    \
	    -u $$(id -u):$$(id -g)                                  \
	    -v $$(pwd):/src                                         \
	    -w /src                                                 \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin                \
	    -v $$(pwd)/.go/bin/$(OS)_$(ARCH):/go/bin/$(OS)_$(ARCH)  \
	    -v $$(pwd)/.go/cache:/.cache                            \
	    --env HTTP_PROXY=$(HTTP_PROXY)                          \
	    --env HTTPS_PROXY=$(HTTPS_PROXY)                        \
	    --env GOFLAGS="-mod=vendor"                             \
	    $(BUILD_IMAGE)                                          \
	    golangci-lint run --enable $(ADDTL_LINTERS) --timeout=10m --skip-files="generated.*\.go$\" --skip-dirs-use-default --skip-dirs=client,vendor

$(BUILD_DIRS):
	@mkdir -p $@

KUBE_NAMESPACE    ?= kubeops
REGISTRY_SECRET   ?=
IMAGE_PULL_POLICY	?= IfNotPresent

ifeq ($(strip $(REGISTRY_SECRET)),)
	IMAGE_PULL_SECRETS =
else
	IMAGE_PULL_SECRETS = --set imagePullSecrets[0].name=$(REGISTRY_SECRET)
endif
>>>>>>> ced7bd5c (Fix certify make target)

.PHONY: install
install:
	$(GO) install -o $(TRICKSTER) $(TAGVER)

# Minikube and helm bootstrapping are done via deploy/helm/Makefile
.PHONY: helm-local
helm-local:
	kubectl config use-context minikube --namespace=trickster
	kubectl scale --replicas=0 deployment/dev-trickster -n trickster
	eval $$(minikube docker-env) \
		&& docker build -f deploy/Dockerfile -t trickster:dev .
	kubectl set image deployment/dev-trickster trickster=trickster:dev -n trickster
	kubectl scale --replicas=1 deployment/dev-trickster -n trickster

# Minikube and helm bootstrapping are done via deploy/kube/Makefile
.PHONY: kube-local
kube-local:
	kubectl config use-context minikube
	kubectl scale --replicas=0 deployment/trickster
	eval $$(minikube docker-env) \
		&& docker build -f deploy/Dockerfile -t trickster:dev .
	kubectl set image deployment/trickster trickster=trickster:dev
	kubectl scale --replicas=1 deployment/trickster

DOCKER_TARGET ?= final
.PHONY: docker
docker:
	docker buildx build \
		--progress=plain \
		--build-arg IMAGE_ARCH=$(IMAGE_ARCH) \
		--build-arg GIT_LATEST_COMMIT_ID=$(GIT_LATEST_COMMIT_ID) \
		--target $(DOCKER_TARGET) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg TAGVER=$(TAGVER) \
		-f ./Dockerfile \
		-t trickster:$(TAGVER) \
		--platform linux/$(IMAGE_ARCH) \
		.

.PHONY: docker-release
docker-release:
# linux x86 image
	docker build --build-arg IMAGE_ARCH=amd64 --build-arg GOARCH=amd64 -f ./deploy/Dockerfile -t trickstercache/trickster:$(IMAGE_TAG) .
# linux arm image
	docker build --build-arg IMAGE_ARCH=arm64v8 --build-arg GOARCH=arm64 -f ./deploy/Dockerfile -t trickstercache/trickster:arm64v8-$(IMAGE_TAG) .

.PHONY: style
style:
	! gofmt -d $$(find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

LINT_FLAGS ?= 
.PHONY: lint
lint:
	@go tool golangci-lint run $(LINT_FLAGS) -c .golangci.yml

.PHONY: lint-fix
lint-fix:
	@LINT_FLAGS="--fix" $(MAKE) lint
	@go tool golangci-lint fmt -c .golangci.yml

GO_TEST_FLAGS ?= -coverprofile=.coverprofile
.PHONY: test
test: check-license-headers check-codegen gotest check-fmtprints check-todos

.PHONY: gotest
gotest:
	go test -timeout=5m -v ${GO_TEST_FLAGS} ./...

.PHONY: data-race-test
data-race-test:
	GO_TEST_FLAGS="-race" $(MAKE) test | tee race-output.log

.PHONY: data-race-test-inspect
data-race-test-inspect:
	./hack/inspect-race-output.sh race-output.log

.PHONY: bench
bench:
	bash -c "$(GO) test -v -coverprofile=.coverprofile ./... -run=nonthingplease -bench=. | grep -v ' app=trickster '; exit ${PIPESTATUS[0]}"

.PHONY: test-cover
test-cover: test
	$(GO) tool cover -html=.coverprofile

.PHONY: clean
clean:
	rm -rf ./trickster ./$(BUILD_SUBDIR)

.PHONY: generate
generate: perform-generate insert-license-headers

.PHONY: perform-generate
perform-generate:
	$(GO) generate ./pkg/... ./cmd/...

.PHONY: insert-license-headers
insert-license-headers:
	@for file in $$(find ./pkg ./cmd -name '*.go') ; \
	do \
		output=$$(grep 'Licensed under the Apache License' $$file) ; \
		if [ "$$?" != "0" ]; then \
			echo "adding License Header Block to $$file" ; \
			cat $(BUMPER_FILE) > /tmp/trktmp.go ; \
			cat $$file >> /tmp/trktmp.go ; \
			mv /tmp/trktmp.go $$file ; \
		fi ; \
	done

CODEGEN_PATHS ?= "'./pkg/**_gen.go'"
.PHONY: check-codegen
check-codegen:
	@$(MAKE) generate > /dev/null
	@git diff --name-only --exit-code ${CODEGEN_PATHS}

.PHONY: check-license-headers
check-license-headers: SHELL:=/bin/sh
check-license-headers:
	@for file in $$(find ./pkg ./cmd -name '*.go') ; \
	do \
		output=$$(grep 'Licensed under the Apache License' $$file) ; \
		if [ "$$?" != "0" ]; then \
			echo "" ; \
			echo "Some project code files do not have the Trickster / Apache 2.0 license header." ; \
			echo "Run 'make insert-license-headers' and commit the changes." ; \
			echo "" ; \
			exit 1 ; \
		fi ; \
	done ; \
	echo "" ; echo "\033[1;32m✓\033[0m All code files have the required license header." ; echo ""

.PHONY: check-fmtprints
check-fmtprints: SHELL:=/bin/sh
check-fmtprints: # fails if there are any fmt.Print* calls outside of the 3 approved files
	@cd pkg && \
	fmtprints=$$(git grep -n fmt.Print | grep -v 'appinfo/usage/usage.go' | grep -v '^daemon/'); \
	count=0; \
	if [ -n "$$fmtprints" ]; then \
		count="$$(echo "$$fmtprints" | wc -l | tr -d '[:space:]')" ; \
	fi; \
	if [ "$$count" -ne 0 ]; then \
		echo "" ; \
		echo "\033[1;31m⨉\033[0m ($$count) unexpected fmt.Print*(s) must be removed from the codebase:"; \
		echo "" ; \
		echo "$$fmtprints" ; \
		echo "" ; \
		echo "" ; \
		exit 1; \
	fi ; \
	echo "" ; echo "\033[1;32m✓\033[0m No unexpected fmt.Print* calls." ; echo ""

.PHONY: check-todos
check-todos: SHELL:=/bin/sh
check-todos: # there are 11 known "TODO"s in the codebase. This check fails if more are added.
	@cd pkg && \
	todos=$$(git grep -in todo | grep -v 'context\.TODO'); \
	count=0; \
	if [ -n "$$todos" ]; then \
		count="$$(echo "$$todos" | wc -l | tr -d '[:space:]')" ; \
	fi; \
	KNOWN_TODO_COUNT=7 ; \
	if [ "$$count" -gt $$KNOWN_TODO_COUNT ]; then \
		newtodos=$$(($$count - $$KNOWN_TODO_COUNT)) ; \
		echo "" ; \
		echo "\033[1;31m$$newtodos new TODOs found in the codebase.\033[0m Do not add any new TODOs to the codebase." ;\
		echo "" ; \
		echo "All TODOs:" ; \
		echo "" ; \
		echo "$$todos" | cut -b 1-100 ; \
		echo "" ; \
		echo "" ; \
		exit 1; \
	fi ; \
	echo "" ; echo "\033[1;32m✓\033[0m No new TODOs found." ; echo ""

.PHONY: spelling
spelling:
	@which mdspell ; \
	if [ "$$?" != "0" ]; then \
		echo "mdspell is not installed" ; \
	else \
		mdspell './README.md' './docs/**/*.md' ; \
	fi
	@which codespell ; \
	if [ "$$?" != "0" ]; then \
		echo "codespell is not installed" ; \
	else \
		codespell --skip='vendor,*.git,*.png,*.pdf,*.tiff,*.plist,*.pem,rangesim*.go,*.gz,go.sum,go.mod' --ignore-words='./testdata/ignore_words.txt' ; \
	fi

.PHONY: serve
serve:
	@cd cmd/trickster && go run . -config /etc/trickster/trickster.yaml

.PHONY: serve-debug
serve-debug:
	@cd cmd/trickster && go run . -config /etc/trickster/trickster.yaml --log-level debug

.PHONY: serve-info
serve-info:
	@cd cmd/trickster && go run . -config /etc/trickster/trickster.yaml --log-level info

.PHONY: serve-cli
serve-cli:
	@cd cmd/trickster && go run . -origin-url http://127.0.0.1:9090/ -provider prometheus

GOLANG_CI_LINT_VERSION ?= v2.7.2
.PHONY: get-tools
get-tools: get-msgpack
	@echo "Installing tools..."
	go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANG_CI_LINT_VERSION)
	go get -tool honnef.co/go/tools/cmd/staticcheck@2025.1.1

.PHONY: get-msgpack
get-msgpack:
	$(GO) get -tool github.com/tinylib/msgp@$(shell go list -m github.com/tinylib/msgp | cut -d' ' -f2)

.PHONY: developer-start
developer-start:
	@cd docs/developer/environment && docker compose up -d
	
.PHONY: developer-stop
developer-stop:
	@cd docs/developer/environment && docker compose stop

.PHONY: developer-delete
developer-delete:
	@cd docs/developer/environment && docker compose down -v --remove-orphans

.PHONY: developer-recreate
developer-recreate: developer-delete
	@cd docs/developer/environment && docker compose up -d

.PHONY: developer-seed-data
developer-seed-data:
	@cd docs/developer/environment && docker compose run --rm clickhouse_seed

RUN_FLAGS ?=
.PHONY: serve-dev
serve-dev:
	@go run $(RUN_FLAGS) cmd/trickster/main.go -config $(if $(TRK_CONFIG),$(TRK_CONFIG),docs/developer/environment/trickster-config/trickster.yaml)

serve-dev-data-race:
	RUN_FLAGS=-race $(MAKE) serve-dev | tee race-output.log
