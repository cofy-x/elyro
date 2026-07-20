# Changelog

All notable changes to Elyro are documented here. The project follows Semantic Versioning, and pre-1.0 releases may contain intentional breaking changes.

## [0.1.5] - 2026-07-20

Elyro v0.1.5 is the first non-prerelease version and establishes the compatibility baseline for subsequent pre-1.0 updates.

### Added

- Added side-effect-free `elyro up --dry-run` plans with typed create, start, reuse, and recreate actions, stable specification reason codes, resolved project identity, and image availability.
- Added `elyro down --dry-run` plans that distinguish removal, stale managed-state cleanup, and empty state while explicitly reporting preserved host data.

### Changed

- Actual `up --json` results now include the same stable reasons used by lifecycle planning, and recreated human receipts explain why replacement occurred.
- Workspace specification comparison is shared by planning, execution, and Doctor diagnostics so previewed and applied actions follow one contract.
- Release publishing now marks ordinary semantic-version tags as production-ready and latest while retaining automatic prerelease behavior for future suffixed tags.

### Fixed

- Stopped Workspace startup now preserves SSH trust when Docker reassigns the loopback host port but the container host keys remain unchanged.
- Empty `down` cleanup no longer creates blank managed SSH or known-host files.

## [0.1.4] - 2026-07-20

### Added

- Added strict `docker.environment` and ordered project-relative `docker.env_files` inputs for reproducible container-wide runtime configuration.
- Added schema-2 Doctor metadata and a stable `runtime_environment` check that expose variable names and relative file paths while redacting values and fingerprints.
- Added a public runtime-environment example and installed-binary E2E coverage for precedence, redaction, automatic recreation, SSH/editor inheritance, and cleanup.

### Changed

- Workspace specification matching now fingerprints final effective runtime values, so effective changes automatically recreate an existing Workspace while equivalent overrides remain reusable.
- Managed SSH and editor terminals inherit the same explicit runtime values as `exec` and `shell`; values are streamed to container setup without entering host argv, logs, labels, registry data, or public JSON.
- Automatic specification replacement now reports the existing typed `recreated` action instead of `created`.

## [0.1.3] - 2026-07-20

### Added

- Added `elyro image init` to scaffold a project-owned Dockerfile and validated build configuration derived from an official Toolchain image.
- Added `elyro image build` with argv-safe Docker execution, streamed build logs, typed human receipts, and a schema-1 JSON result.

### Changed

- `elyro up` and schema-2 Doctor diagnostics now distinguish a missing configured project image and direct the user to the explicit build command; ordinary Workspace startup never builds implicitly.
- Documented when to use temporary Workspace packages, language-native project dependencies, derived Workspace images, or external multi-service and Feature tooling.

## [0.1.2] - 2026-07-19

### Changed

- Refined terminal output with a cobalt-blue brand and section palette, clearer semantic roles, shorter command descriptions, and a compact first-run flow.
- `elyro init` now focuses only on project detection, target validation, and configuration creation; system diagnostics remain in `elyro doctor`, while runtime checks remain in `elyro up` and `elyro open`.
- `elyro open` now defaults to the first detected editor on Enter and provides explicit, successful cancellation with `q` or `cancel`.
- Workspace startup progress now describes user-visible preparation, image pulling, and startup without exposing SSH, registry, or container implementation phases.
- Official Toolchain images now provide a native zsh prompt that shows the Workspace and current Linux directory, plus color-aware listings, autosuggestions, and syntax highlighting without a shell framework.

### Fixed

- `elyro shell` now forwards only supported color overrides instead of inheriting host-specific terminal capabilities that may be unavailable in the Workspace.
- Official images no longer export `DEBIAN_FRONTEND=noninteractive` at runtime; apt remains non-interactive only while images are built.

## [0.1.1] - 2026-07-19

### Changed

- Project-scoped commands now resolve the nearest `elyro.yaml`, then a containing registered Workspace, then the nearest Git root, so subdirectory commands share one deliberate project identity.
- `elyro doctor` now provides project-aware grouped diagnostics and a schema-2 JSON contract with structured project resolution and scoped checks.
- `elyro up --recreate` now replaces a running or stopped Workspace after completing all non-destructive preflight checks.
- Added the complete `elyro.yaml` configuration reference and documented selection, precedence, safety, editor, port, mount, and custom-image behavior.

### Fixed

- Prevented commands run from nested project directories from reporting or creating a different accidental Workspace.
- Added an explicit recovery path for custom images rebuilt under an unchanged tag.

## [0.1.0] - 2026-07-18

The first Elyro release establishes a Mac-first, local-first Linux Workspace for individual developers and host coding agents.

### Added

- Added the `elyro` CLI for Workspace initialization, lifecycle, status, command execution, shell access, editor handoff, diagnostics, version reporting, and embedded Skill installation.
- Added zero-configuration Toolchain detection for Go, Python, Node.js, and Java projects, plus explicit `elyro.yaml` environments and custom-image support.
- Added schema-stable JSON output for automation and a host-agent Skill for Codex and Claude Code.
- Added five maintained dual-architecture Workspace images: base, Python, Go, Node.js, and Java.
- Added exact-version macOS and Linux release archives, checksums, provenance attestations, Homebrew publishing, and installation smoke tests.
- Added immutable dual-architecture candidate images with architecture, OCI metadata, and compressed-size budget validation before release.

### Security

- Workspace SSH identities and known-host records are Elyro-managed and isolated from the user's global SSH trust files.
- Published image inputs and Toolchain archives are version- and digest-pinned.
- Elyro remains a development environment rather than a security sandbox and does not install, authenticate, run, or proxy coding agents.
