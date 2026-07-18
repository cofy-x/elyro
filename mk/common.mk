# Shared helper macros for Make recipes.

define require-var
	@if [ -z "$($(1))" ]; then \
		echo "Error: $(1) is required. Example: $(2)"; \
		exit 1; \
	fi
endef

define require-file
	@if [ ! -f "$(1)" ]; then \
		echo "Error: $(1) not found"; \
		exit 1; \
	fi
endef

define require-image
	@docker image inspect "$(1)" >/dev/null 2>&1 || { \
		echo "Missing image $(1). Run 'make $(2)' first."; \
		exit 1; \
	}
endef
