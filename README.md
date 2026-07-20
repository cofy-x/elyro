# Elyro

[![Release](https://img.shields.io/github/v/tag/cofy-x/elyro?sort=semver&label=release)](https://github.com/cofy-x/elyro/releases)
[![CI](https://github.com/cofy-x/elyro/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/cofy-x/elyro/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/cofy-x/elyro)](LICENSE)

Edit on Mac. Build and test in Linux.

Elyro gives people and coding agents a predictable local Linux Workspace. Your project stays on the Mac; build, test, and debug commands run in a maintained Linux container; VS Code or Cursor can take over through Remote SSH.

```bash
elyro up --dry-run
elyro up --open
elyro shell
elyro exec -- go test ./...
```

![Elyro terminal workflow](docs/assets/elyro-demo.gif)

Elyro is also designed as a stable execution tool for a host coding agent:

```bash
elyro skill install codex
elyro status --json
elyro up --json --toolchain go
elyro exec -- go test ./...
```

Elyro does not install, authenticate, run, or proxy a coding agent. The bundled `use-elyro-workspace` Skill teaches an already installed Codex or Claude Code session to use Elyro's machine-readable CLI.

## Why Elyro

- Mac-first and local-first: source files and editing stay on the host.
- Zero-config startup when one Toolchain can be detected; only explicit `init` commands write project configuration.
- Maintained Workspace images for Python, Go, Node.js, and Java, with explicit project images for persistent OS tools and reproducible runtime environment inputs.
- Direct, argv-safe Linux execution through `elyro exec`; shell syntax is explicit with `bash -lc`.
- Editor handoff over managed Remote SSH with strict, isolated host-key trust.
- Small machine contracts for automation: schema 1 for Workspace lifecycle and inspection, plus scoped schema-2 diagnostics.
- Side-effect-free lifecycle plans explain whether `up` will create, start, reuse, or recreate a Workspace and what `down` will remove or preserve.
- Calm terminal receipts with useful next steps for people, while pipes, CI, and JSON remain stable for agents.

## Start

Install from Homebrew:

```bash
brew install cofy-x/tap/elyro
```

v0.1.5 is Elyro's first non-prerelease version. The CLI, `elyro.yaml` version 1, JSON schemas, exit codes, and lifecycle meanings form the compatibility baseline for later pre-1.0 releases.

Then enter a Go, Python, Node.js, or Java project and run:

```bash
elyro up --open
```

Elyro detects a single Toolchain automatically. In a non-interactive or ambiguous project, select one explicitly:

```bash
elyro up --toolchain go
```

Use `elyro init` only when the project needs named Environments, ports, mounts, editor settings, or a custom image.

When a project needs an OS library, compiler, database client, or global CLI, derive a project-owned image from its official Toolchain image. Elyro creates the safe build configuration, while the project keeps ownership of its Dockerfile:

```bash
elyro image init
# edit .elyro/Dockerfile
elyro image build
elyro up --recreate
```

Ordinary `elyro up` never builds a Dockerfile or runs project lifecycle hooks.

Preview lifecycle changes before applying them:

```bash
elyro up --dry-run
elyro down --dry-run
```

Recreating or removing a Workspace discards its container writable layer. Project files, mounted host data, and local images remain on the host; persistent OS tools belong in the project image.

Declare non-sensitive runtime values explicitly when every shell, command, and editor terminal should share them:

```yaml
docker:
  environment:
    APP_ENV: development
  env_files:
    - .elyro/dev.env
```

Elyro never reads `.env` implicitly or inherits arbitrary host variables. Runtime values remain visible through Docker and are not a secret store. See the [runtime environment example](examples/workspace/go-runtime-environment/README.md).

## Coding agents

Install the same embedded Skill for either supported host agent:

```bash
elyro skill install codex
elyro skill install claude-code
# or: elyro skill install all
```

See [Using Elyro with Coding Agents](docs/coding-agents.md) for the operating contract and an `AGENTS.md` snippet.

## Scope

An Elyro Workspace is a development environment, not a security sandbox. Elyro does not replace Docker's security model or the host coding agent's permission and approval system. Elyro intentionally does not provide remote Workspace orchestration, multi-tenancy, enterprise policy, a package manager, background jobs, an agent runtime, or a web UI.

## Documentation

- [Installation](docs/installation.md)
- [Why Elyro](docs/why-elyro.md)
- [Workspace guide](docs/workspace/README.md)
- [Workspace configuration](docs/workspace/configuration.md)
- [CLI reference](docs/workspace/cli-reference.md)
- [Custom images](docs/workspace/custom-images.md)
- [Coding agents](docs/coding-agents.md)
- [Supported images](images/README.md)
- [Roadmap](docs/roadmap.md)

## Development

```bash
go test ./...
make workspace-smoke
make workspace-e2e
make ci-pr-smoke
```

The terminal demo is generated from [`scripts/demo/elyro.tape`](scripts/demo/elyro.tape) with `make demo-record`; it uses a real local Workspace rather than mocked output.

Elyro is licensed under Apache-2.0. Product names and trademarks belong to their respective owners; interoperability does not imply endorsement.
