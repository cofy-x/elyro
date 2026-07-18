# Changelog

All notable changes to Elyro are documented here. The project follows Semantic Versioning, and pre-1.0 releases may contain intentional breaking changes.

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
