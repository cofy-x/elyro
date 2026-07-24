---
name: use-elyro-workspace
description: Use Elyro to build, test, and debug a configured project inside its local Linux Workspace. Use only when the intended project directory contains an elyro.yaml file, or the current working directory is nested below the nearest ancestor that contains one. Do not use merely because the host is macOS, Linux validation is useful, Elyro is installed, a Workspace exists, or agent instructions mention Elyro without identifying a configured project.
---

# Use Elyro Workspace

Use Elyro as the only interface to the project's local Linux development environment.

1. Resolve the intended project directory from the task. Confirm that `elyro.yaml` exists at that directory, or use the nearest ancestor containing it when working from a nested directory. Do not treat a configuration found only in an unrelated descendant as evidence that the current project uses Elyro. If no qualifying `elyro.yaml` exists, stop using this Skill and do not run any `elyro` command.
2. Use `elyro <command> --help` when exact syntax or available options are needed. Treat this Skill as the cross-command workflow and safety contract rather than a duplicate CLI reference.
3. Inspect the current state with `elyro status --json`.
4. Before changing Workspace state, preview the operation with `elyro up --dry-run --json`. A `reuse` plan needs no `up`; a `create` plan may be applied with `elyro up --json`. Ask the user before applying a `start` or `recreate` plan to an existing Workspace.
5. If an existing build configuration reports that its project image is missing, run `elyro image build --json`, then preview again. Do not run `elyro init` or `elyro image init` from this Skill; configuration creation is a separate, explicit user action.
6. Run Linux build, test, lint, and debugging commands with `elyro exec -- COMMAND [ARG...]`.
7. For pipes, redirects, expansion, or compound shell syntax, invoke it explicitly with `elyro exec -- bash -lc '...'`.
8. If the environment is unhealthy or a prerequisite is missing, inspect `elyro doctor --json` before proposing changes.

Project `docker.environment` and `docker.env_files` values are inherited by every Workspace command. For a temporary command-only override, use `elyro exec -- env KEY=value COMMAND`; do not rewrite project configuration or assume Elyro provides secret storage.

Do not bypass Elyro by invoking Docker or SSH directly. Do not install or run a coding-agent CLI inside the Workspace. Do not stop, recreate, or remove a Workspace that existed before the task without user approval. Leave `elyro open` and destructive cleanup such as `elyro down` to the user unless they explicitly request it; `elyro down --dry-run --json` may be used for a read-only explanation.
