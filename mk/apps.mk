.PHONY: elyro-build elyro-install elyro-smoke release-config-check release-install-smoke \
	workspace-build workspace-install workspace-smoke workspace-e2e \
	test ci-pr-smoke ci-nightly ci-weekly ci-suite

ELYRO_BIN_DIR := bin
ELYRO_BIN := $(ELYRO_BIN_DIR)/elyro
ELYRO_INSTALL_DIR ?=
WORKSPACE_SMOKE_DIR := scripts/workspace/smoke
ELYRO_CI_SUITE_MODE ?= pr
ELYRO_VERSION ?= dev
ELYRO_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf unknown)
ELYRO_BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
ELYRO_LDFLAGS := -s -w \
	-X github.com/cofy-x/elyro/internal/version.Version=$(ELYRO_VERSION) \
	-X github.com/cofy-x/elyro/internal/version.Commit=$(ELYRO_COMMIT) \
	-X github.com/cofy-x/elyro/internal/version.BuildDate=$(ELYRO_BUILD_DATE)

elyro-build:
	@mkdir -p "$(ELYRO_BIN_DIR)"
	@go build -ldflags "$(ELYRO_LDFLAGS)" -o "$(ELYRO_BIN)" ./cmd/elyro

elyro-install:
	@if [ -n "$(ELYRO_INSTALL_DIR)" ]; then GOBIN="$(ELYRO_INSTALL_DIR)" go install ./cmd/elyro; else go install ./cmd/elyro; fi

elyro-smoke: workspace-smoke

release-install-smoke:
	@"scripts/ci/install-release-smoke.sh"

release-config-check:
	@goreleaser check
	@"scripts/ci/check-release-inputs.sh"
	@"scripts/ci/candidate-images-test.sh"

test:
	@go test ./cmd/... ./internal/... ./skills/...

workspace-build: elyro-build

workspace-install: elyro-install

workspace-smoke:
	@"$(WORKSPACE_SMOKE_DIR)/verify.sh"

workspace-e2e:
	@"scripts/workspace-e2e.sh"

ci-pr-smoke:
	@"scripts/ci/pr-smoke.sh"

ci-nightly:
	@"scripts/ci/nightly.sh"

ci-weekly:
	@"scripts/ci/weekly.sh"

ci-suite:
	@"scripts/ci/run-suite.sh" "$(ELYRO_CI_SUITE_MODE)"
