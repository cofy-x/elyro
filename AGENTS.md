# AGENTS.md

## Purpose

Elyro is a Mac-first, local-first Linux Workspace tool for individual developers and host coding agents. Elyro does not install or run coding agents; it provides a stable CLI and embedded Skill for Linux project execution.

## Context Loading

1. Always read the [repository context index](.x/README.md), then load only the row that matches the task.
2. Keep agent-facing context centralized in this file and `.x/`; route domain context from `.x/README.md` instead of adding subtree `AGENTS.md` files.
3. Read user-facing guides only when the task changes or depends on the documented behavior. Do not preload every document under `.x/` or `docs/`.
4. Product direction and scope decisions require the [product roadmap](docs/roadmap.md).

## Repository-Wide Boundaries

- `cmd/elyro` owns the unified user-facing CLI entrypoint and command-tree wiring.
- `internal/cliui` owns the shared human-facing terminal presentation; it must not alter JSON or command-stream contracts.
- `internal/workspace` owns Workspace configuration, identity, registry, lifecycle, SSH access, and editor handoff.
- `internal/images` owns shared image reference resolution, while `images/` owns reusable image definitions.
- `skills/use-elyro-workspace` is the canonical embedded Skill source used by the `elyro skill` command.

Detailed path ownership and structural sync points live in the [repository ownership map](.x/project-overview.md). Cross-area invariants live in the [integration topology and routing guide](.x/integration-stack.md).

## Design Policy

- Optimize product decisions for the individual developer's local workflow. Do not introduce team orchestration, multi-tenant infrastructure, remote workspace lifecycle, or enterprise abstractions without an explicitly validated product requirement; use the [product roadmap](docs/roadmap.md) as the source of truth for positioning and scope.
- Prefer the clearer long-term model over compatibility layers, transitional aliases, or workaround paths unless an existing user-facing command depends on them or the user explicitly requests them.
- Keep Elyro as a Workspace tool. Coding-agent integration belongs in the host-facing Skill and machine-readable CLI, not an agent runtime, model proxy, or credential layer.
- Prefer cohesive boundary changes when a workflow crosses command-tree, image, or Make ownership, while keeping each change small enough to verify locally.
- Add abstractions only when they clarify an enforced boundary or remove repeated workflow code.
- Keep local machine assumptions, credentials, generated state, and proprietary third-party binaries out of the repository.

## Change Discipline

- Place implementation in its owning area; do not create a shared package merely to avoid choosing an owner.
- Preserve unrelated working-tree changes.
- Update tests and the owning user documentation when behavior changes.
- Before completion, follow the [validation matrix](.x/validation.md) and run the narrowest sufficient checks for every affected area.
- Do not require credentialed live tests by default; report them as skipped when credentials were not supplied.
- Do not commit, push, create tags, publish releases, or change external services unless the user explicitly requests that operation.
