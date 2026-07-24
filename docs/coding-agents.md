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

In a configured project without an installed Skill, a host agent that discovers `elyro.yaml` can inspect the `Agent` section of `elyro --help`, use `elyro skill --help` to find the inspection and installation paths, and run `elyro skill show` for the complete version-matched Skill. After the Skill activates, individual command help remains the source for exact syntax and available options.

The Skill applies only to a configured project: `elyro.yaml` must exist in the intended project directory or the nearest ancestor of the current working directory. This activation boundary applies to the Skill; it does not remove Elyro's zero-configuration CLI workflow for direct human use. The agent does not infer Elyro usage merely from macOS, a need for Linux validation, an installed Elyro binary, an existing Workspace, or a generic mention in agent instructions. It also does not scan unrelated descendants for configuration. If the precondition is met, the Skill directs the host agent to inspect `elyro status --json`, preview lifecycle changes with `elyro up --dry-run --json`, start a missing Workspace only when needed, and run Linux commands through `elyro exec -- ...`. A `recreate` plan is surfaced to the user instead of being applied implicitly by the agent. Project runtime environment configuration is inherited automatically; a temporary per-command override can use `elyro exec -- env KEY=value command`. The Skill does not contain scripts, credentials, MCP configuration, or model settings.

## AGENTS.md snippet

```markdown
## Linux validation

Use the installed `use-elyro-workspace` Skill only when this project's root contains `elyro.yaml`; when working in a nested directory, use the nearest ancestor containing that file. Otherwise, do not use the Skill or run Elyro commands. For a configured project, inspect `elyro status --json` first and preview changes with `elyro up --dry-run --json`. Start a missing Workspace with `elyro up --json`; ask before applying a `start` or `recreate` plan to an existing Workspace. Run commands with `elyro exec -- ...`; use `elyro exec -- bash -lc '...'` only for explicit shell syntax. Do not call Docker or SSH directly, do not create Elyro configuration from the Skill, and leave `elyro down` to the user.
```

## Safety boundary

A Workspace is a convenient development environment, not a security sandbox. Elyro does not proxy model traffic, manage agent credentials, or replace the coding agent's permission model. Editor opening and destructive cleanup remain user decisions.
