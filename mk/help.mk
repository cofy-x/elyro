.PHONY: help list-targets

# Usage:
#   make help
help:
	@echo "elyro Make targets (grouped)"
	@echo ""
	@echo "User workflows:"
	@echo "  make elyro-build"
	@echo "  make elyro-install"
	@echo "  make workspace-build"
	@echo "  make demo-record"
	@echo ""
	@echo "Validation:"
	@echo "  make workspace-smoke"
	@echo "  make workspace-e2e"
	@echo "  make release-config-check"
	@echo "  make release-install-smoke"
	@echo "  make test"
	@echo "  make ci-pr-smoke"
	@echo "  make ci-nightly"
	@echo "  make ci-weekly"
	@echo "  make ci-suite ELYRO_CI_SUITE_MODE=pr|nightly|weekly|all"
	@echo ""
	@echo "Workspace images:"
	@echo "  make image-report VERSION=v0.1.6 [COMPARE_VERSION=<previous-version>]"
	@echo "  make workspace-base-image-build"
	@echo "  make workspace-python-image-build"
	@echo "  make workspace-go-image-build"
	@echo "  make workspace-node-image-build"
	@echo ""
	@echo "All targets (alphabetical):"
	@$(MAKE) list-targets

# Usage:
#   make list-targets
list-targets:
	@awk -F: '/^[a-zA-Z0-9][a-zA-Z0-9_.-]*:/{print $$1}' $(MAKEFILE_LIST) | sort -u
