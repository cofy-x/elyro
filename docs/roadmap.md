# Elyro Roadmap

Elyro is a Mac-first, local-first Linux Workspace for individual developers and host coding agents. Project files and editing stay on macOS; Linux execution happens through a small, stable CLI.

## Product model

| Concept | Meaning |
| :--- | :--- |
| Workspace | Local Linux development environment for one host project |
| Environment | Named project configuration in `elyro.yaml` |
| Toolchain | Maintained Python, Go, Node.js, or Java Workspace image |
| Project image | Project-owned Dockerfile derived from a Toolchain image and built explicitly through Elyro |
| Runtime environment | Explicit non-secret container variables resolved from project configuration and project-relative files |
| Skill | Guidance that teaches a host coding agent to use the Elyro CLI |

## Current priorities

- Keep `elyro up --open` a verifiable one-command human path.
- Keep JSON output stable and stdout clean for coding-agent automation.
- Keep the human CLI recognizable and calm with semantic terminal hierarchy, without adding a full-screen UI or changing stream behavior.
- Preserve direct argv execution, exit codes, stdin, signals, project workdir, and predictable hostname.
- Keep SSH limited to editor handoff and direct user access, with strict isolated host-key trust.
- Maintain five amd64/arm64 images: `workspace-base`, `workspace-python`, `workspace-go`, `workspace-node`, and `workspace-java`.
- Keep official Toolchain images limited to the language runtime, primary build or dependency-management tools, and native compilation prerequisites; project-level linters and formatters stay project-owned.
- Keep persistent project OS tools in an explicit project-owned Dockerfile; never turn ordinary Workspace startup into an implicit package installation or lifecycle-hook runner.
- Keep runtime configuration explicit and reproducible; never inherit arbitrary host variables, load `.env` implicitly, or present Docker-visible values as secrets.
- Measure cold and warm startup and image size before adding image variants, and reject release candidates that exceed reviewed compressed-size budgets.

## Explicit non-goals

- running, authenticating, or proxying coding agents
- remote Workspace lifecycle, teams, multi-tenancy, or enterprise governance
- Windows, web UI, background task queues, or package management
- animated or full-screen terminal interfaces that obscure ordinary command output
- Dev Container compatibility or a template marketplace
- secret storage, host-environment inheritance, variable interpolation, or lifecycle hooks

Every public concept must use the same name in CLI, YAML, JSON, images, Make targets, examples, and documentation.
