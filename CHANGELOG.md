# Changelog

All notable changes to Elyro are documented here. Versions through v0.7.0 were
released under the Runx name; v0.8.0 is the clean-break rename to Elyro. The
project follows Semantic Versioning, and pre-1.0 releases may contain
intentional breaking changes.

## [0.9.0] - 2026-07-18

Elyro narrows the maintained Toolchain contract and makes published image size a tag-before-release decision instead of a post-release observation.

### Added

- Added immutable, commit-bound, dual-architecture candidate images to manually dispatched Nightly runs.
- Added candidate and published-image gates for architecture completeness, OCI revision labels, compressed-size budgets, and comparison with the reviewed previous version.
- Added a real offline `node-gyp` native-addon build to the Node.js image smoke contract.

### Changed

- Reduced the Node.js native build environment to `g++`, `make`, Python, bundled Node.js headers, and npm's bundled `node-gyp` instead of the broader `build-essential` package set.
- Generalized the release documentation and made image budgets reviewed release inputs.

### Removed

- Removed `golangci-lint` from `workspace-go`; optional linters and formatters are now explicitly project-owned or custom-image concerns.

## [0.8.0] - 2026-07-18

Elyro establishes a smaller, product-first contract for local Linux development from macOS and host coding agents.

### Added

- Added the maintained `workspace-node` image with pinned Node.js LTS, npm, Corepack, and native-addon build dependencies.
- Added schema-1 Workspace JSON views that expose stable product concepts without leaking Docker or SSH implementation identifiers.

### Changed

- Renamed the product, binary, configuration, environment variables, image namespace, embedded Skill, and public repository identity to Elyro.
- Reduced the public CLI to Workspace lifecycle, project initialization, diagnostics, and Skill installation.
- Made the product promise explicit: edit on Mac, then build and test in Linux.
- Reduced the Go image layer size by keeping compiler and module caches out of the published layer.

### Removed

- Removed project scaffolding, `current`, and public image-build commands.
- Removed public container, label, SSH alias, identity, and known-hosts details from JSON output.

## [0.7.0] - 2026-07-18

Runx now presents a clearer terminal workflow while preserving its automation-first contracts.

### Added

- Added a small shared Lip Gloss presentation layer with semantic success, warning, failure, progress, field, and next-step styles.
- Added grouped top-level help and concise completion receipts for Workspace, project, image, Skill, doctor, and version commands.
- Added a reproducible VHS terminal demo backed by real Runx commands.

### Changed

- Human output now uses restrained color only on capable terminals and remains plain for redirected output, `NO_COLOR`, `TERM=dumb`, and CI.
- Interactive decisions now derive terminal capability from each command's explicit input and output streams, improving tests and embedding behavior.

### Preserved

- JSON schemas and stdout purity, command exit codes, `exec` and `shell` streams, and byte-exact `skill show` output remain unchanged.

## [0.6.0] - 2026-07-18

Runx now focuses exclusively on local Linux Workspaces for people and host coding agents.

### Added

- Added schema-stable JSON output for `up`, `down`, and `doctor`, including lifecycle action, duration, hostname, and structured checks.
- Added the embedded `use-runx-workspace` Skill with safe, idempotent Codex and Claude Code installation commands.
- Added strict Workspace-specific SSH known-host management for editor and direct SSH access.

### Changed

- `shell` and `exec` now use local `docker exec` as user `runx` in the project directory, preserving argv, standard streams, signals, and exit codes without depending on SSH.
- Workspace containers now use the sanitized project name as hostname and rebuild when the hostname specification changes.
- The image and release matrix now contains only `workspace-base`, `workspace-python`, `workspace-go`, and `workspace-java`; release archives contain only `runx`.

### Removed

- Removed the integrated Agent runtime, manager, gateway, provider, daemon, model proxy, credentials, and all Agent commands and images.
- Removed Workspace AI CLI commands and bundled coding-agent CLIs.

## [0.5.0] - 2026-07-17

Runx now presents one Workspace-first product model. v0.5.0 defines a new contract and contains no compatibility readers, migration commands, old-state discovery, command aliases, or field translation logic.

### Added

- Added zero-write `runx up`, explicit `runx up --open [--editor cursor|code]`, strict `runx.yaml`, and Workspace schema-2 status/registry output.
- Added a local-only Agent control plane with Provider configuration, strict managed SSH access, and synchronous `agent exec` automation.

### Changed

- Unified Workspace implementation, configuration, registry, access, editor, runtime, and scaffold ownership under `internal/workspace` with Environment and Toolchain terminology.
- Split language Toolchains into their Workspace images. `runtime-base` now contains only the shared Linux, SSH, process, shell, Git, curl, jq, Node.js, and npm contract.
- Python Workspace development now uses Python `venv` and a checksummed standalone `uv` installation.

### Removed

- Removed all Workspace and Agent dashboards, browser terminals, remote Node routing, and public remote-manager controls.
- Removed compatibility code and historical command, configuration, state, image, and Make aliases.
- Removed Poetry, pipx, pnpm, nginx, and language Toolchains from `runtime-base`.

## [0.4.0] - 2026-07-17

Beta release closing the standalone Agent public execution workflow and adding
repeatable, pull-free release image size reporting.

### Added

- Added `runx agent exec` for synchronous, non-interactive commands with
  streamed standard I/O, remote exit codes, optional timeouts, and remote
  process-tree cleanup.
- Added authenticated per-instance SSH access preparation shared by the CLI
  and Dashboard terminal, with dedicated identities and strict managed
  `known_hosts` files.
- Added `make image-report` for inspecting pinned release image manifests,
  dual-architecture compressed sizes, version deltas, and largest layers
  without pulling images into the local Docker cache.

### Changed

- Agent automation documentation now directs non-interactive workflows to
  `agent exec`, while `agent ssh` and `agent shell` remain interactive entry
  points.
- Credentialed Codex live validation now exercises the public `agent exec`
  path instead of an internal container execution workaround.

### Fixed

- Manager status, stop, restart, and bulk instance operations now consistently
  authenticate with the manager token, including custom state and token-file
  locations.
- Agent SSH access now rejects unexpected host-key changes for an existing
  container and refreshes managed host keys only after container replacement.

## [0.3.0] - 2026-07-17

Beta release adding a small, explicit contract for fully project-owned Devbox
images without expanding Runx into an image build system.

### Added

- Added a distro-neutral contract and verified Ubuntu 24.04 from-scratch
  example for project-owned Devbox images that do not inherit from Runx images.

### Changed

- Devbox management now installs SSH access and optional Codex authentication
  as UID 0, allowing compatible images to keep a non-root default user.
- Images explicitly configured in `devbox.yaml` remain user-owned and must be
  built or pulled before `runx devbox up`, even when a profile also sets a
  built-in flavor for editor defaults.

## [0.2.0] - 2026-07-16

Beta release focused on making the first five minutes with Devbox clear for
both existing projects and new dependency-free starters.

### Added

- Added project-aware `runx init` and `runx devbox init` with Python, Go, and
  Java detection and explicit non-interactive write confirmation.
- Added embedded dependency-free `python-http` and `go-http` starters through
  `runx new`.
- Added `runx devbox shell` and exit-code-preserving `runx devbox exec`.
- Added strict `docker.publish` project-profile configuration for loopback-only
  port mappings, including template defaults.

### Changed

- Devbox startup now streams pull and lifecycle progress, prints a concise
  next-step summary, and leaves editor opening to `runx devbox open`.
- `runx init` no longer installs Agent profiles; use
  `runx agent profile init` for that independent workflow.
- Image builds now distinguish an unset `RUNX_PROXY_URL` (automatic detection)
  from an explicit empty value (proxy disabled), as documented.
- Devbox custom-profile smoke builds follow the requested host architecture
  instead of forcing amd64.

## [0.1.1] - 2026-07-16

Maintenance beta release improving package-manager distribution and published
image traceability.

### Added

- Added OCI source, version, revision, and license metadata to published core
  images for reliable GHCR repository linking and source traceability.
- Added generated Homebrew/Linuxbrew Formula publishing, local and published
  installation gates, and a complete public installation guide.

## [0.1.0] - 2026-07-15

First public beta of the Runx CLI, devbox workflow, agent workflow, and core
runtime images for macOS/Linux amd64 and arm64.

### Added

- Apache-2.0 open-source project baseline.
- Public security, contribution, conduct, and third-party notice policies.
- Local-only authenticated Manager control plane and explicit unsafe-profile
  opt-ins.
- Standalone embedded Dashboard, version metadata, and exact GHCR image
  resolution.
- Exact-version macOS/Linux release installer with SHA-256 verification and
  archive installation smoke coverage.
- Pinned build inputs plus PR, nightly, and signed prerelease workflows.

### Fixed

- Hardened arm64 Ubuntu package resolution with strict apt failures, bounded
  retries, and an official Ubuntu Ports fallback.
