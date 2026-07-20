---
name: use-elyro-workspace
description: Use Elyro to build, test, and debug the current project inside its local Linux Workspace. Apply when a project uses Elyro, when Linux-specific validation is needed from macOS, or when AGENTS.md asks the coding agent to execute development commands through Elyro.
---

# Use Elyro Workspace

Use Elyro as the only interface to the project's local Linux development environment.

1. Inspect the current state with `elyro status --json`.
2. Before changing Workspace state, preview the operation with `elyro up --dry-run --json`. A `reuse` plan needs no `up`; a `create` plan may be applied with `elyro up --json`. Ask the user before applying a `start` or `recreate` plan to an existing Workspace.
3. When Toolchain detection is ambiguous, inspect the project and retry the plan with an explicit `--toolchain`; do not run `elyro init` or `elyro image init` unless the user asks to create configuration. If an existing build configuration reports that its project image is missing, run `elyro image build --json`, then preview again.
4. Run Linux build, test, lint, and debugging commands with `elyro exec -- COMMAND [ARG...]`.
5. For pipes, redirects, expansion, or compound shell syntax, invoke it explicitly with `elyro exec -- bash -lc '...'`.
6. If the environment is unhealthy or a prerequisite is missing, inspect `elyro doctor --json` before proposing changes.

Project `docker.environment` and `docker.env_files` values are inherited by every Workspace command. For a temporary command-only override, use `elyro exec -- env KEY=value COMMAND`; do not rewrite project configuration or assume Elyro provides secret storage.

Do not bypass Elyro by invoking Docker or SSH directly. Do not install or run a coding-agent CLI inside the Workspace. Do not stop, recreate, or remove a Workspace that existed before the task without user approval. Leave `elyro open` and destructive cleanup such as `elyro down` to the user unless they explicitly request it; `elyro down --dry-run --json` may be used for a read-only explanation.
