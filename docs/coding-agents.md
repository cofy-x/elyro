# Using Elyro with Coding Agents

Elyro does not run a coding agent. Codex, Claude Code, or another host agent edits the project on macOS and uses Elyro only for Linux Workspace execution.

Install the embedded Skill:

```bash
elyro skill install codex
elyro skill install claude-code
elyro skill install all
```

Inspect it without installing:

```bash
elyro skill show
```

The Skill directs the host agent to inspect `elyro status --json`, preview lifecycle changes with `elyro up --dry-run --json`, start a missing Workspace only when needed, and run Linux commands through `elyro exec -- ...`. A `recreate` plan is surfaced to the user instead of being applied implicitly by the agent. Project runtime environment configuration is inherited automatically; a temporary per-command override can use `elyro exec -- env KEY=value command`. The Skill does not contain scripts, credentials, MCP configuration, or model settings.

## AGENTS.md snippet

```markdown
## Linux validation

Use the installed `use-elyro-workspace` Skill for Linux build, test, lint, and debugging commands. Inspect `elyro status --json` first and preview changes with `elyro up --dry-run --json`. Start a missing Workspace with `elyro up --json` and an explicit `--toolchain` when detection is ambiguous; ask before applying a `start` or `recreate` plan to an existing Workspace. Run commands with `elyro exec -- ...`; use `elyro exec -- bash -lc '...'` only for explicit shell syntax. Do not call Docker or SSH directly, do not run `elyro init` without approval, and leave `elyro down` to the user.
```

## Safety boundary

A Workspace is a convenient development environment, not a security sandbox. Elyro does not proxy model traffic, manage agent credentials, or replace the coding agent's permission model. Editor opening and destructive cleanup remain user decisions.
