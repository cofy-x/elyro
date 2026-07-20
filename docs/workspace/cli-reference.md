# Workspace CLI Reference

```text
elyro init
elyro image init
elyro image build
elyro up
elyro down
elyro shell
elyro exec
elyro open
elyro status
elyro list
elyro doctor
elyro skill
elyro version
```

That is the complete public command surface. Release image maintenance remains a repository maintainer operation; `elyro image` manages only a project's explicitly configured image build.

See the [Workspace configuration reference](configuration.md) for the complete `elyro.yaml` contract.

## Lifecycle

```bash
elyro up [--toolchain python|go|node|java | --environment NAME] [--platform linux/amd64|linux/arm64]
elyro up --open [--editor cursor|code]
elyro up --recreate
elyro up --dry-run [--recreate] [--json]
elyro down
elyro down --dry-run [--json]
```

`--environment` and `--toolchain` are mutually exclusive. `up --json` never prompts and returns schema 1 with `action`, stable `reasons`, `duration_ms`, and a public Workspace view. Runtime environment values, variable names, files, and fingerprints are not added to lifecycle JSON or the registry. `down` removes the container, managed SSH access, and registry entry; `down --json` returns the same Workspace view plus `removed`.

`up --dry-run` uses the real project and Workspace resolver without changing local state. Its schema-1 `workspace_plan` document reports `create`, `start`, `reuse`, or `recreate`, stable reason codes, project-root source, desired Workspace fields, and whether the official image is available or would be pulled. Missing project/custom images and invalid or unsafe inputs still fail. It never pulls, creates an SSH identity, starts or removes a container, or writes managed files. `--dry-run` cannot be combined with `--open` or `--editor`.

`down --dry-run` reports `remove` when a container exists, `cleanup` for stale managed state, and `none` when nothing is owned by Elyro. Its `removes` list covers the container writable layer, managed SSH, known-host trust, and registry record as applicable; `preserves` always identifies project files, mounted host data, and local images. Actual `down` remains non-interactive.

`up` detects a single Toolchain when possible. Ambiguous or unknown projects must choose `--toolchain` in non-interactive use. Only `elyro init` and `elyro image init` write `elyro.yaml`; both are explicit and refuse unsafe or ambiguous writes.

Without an explicit `--project-dir`, project-scoped commands use the nearest `elyro.yaml`, then an existing containing Workspace, then the nearest Git root, and finally the current directory. An explicit path always wins. `up --recreate` replaces an existing Workspace after all configuration, image, mount, publish, managed-state, name-conflict, and safety checks pass; its JSON action is `recreated` with reason `explicit_recreate`.

## Project image

```bash
elyro image init [--environment NAME] [--toolchain go] [--image elyro-local/example:dev] [--yes]
elyro image build [--environment NAME] [--platform linux/arm64] [--pull] [--no-cache] [--json]
```

`image init` creates `.elyro/Dockerfile` and adds an explicit build definition to `elyro.yaml`. It only derives from an official Toolchain image; a fully independent image remains a manual custom-image workflow. Existing files and build definitions are never overwritten. Non-interactive initialization requires `--image`, and any non-interactive write requires `--yes`.

`image build` invokes `docker build` directly with the configured context, Dockerfile, tag, and platform. Docker owns cache decisions. Build logs stream to stderr, while stdout contains only the final human receipt or schema-1 JSON document. A failed build does not touch the existing Workspace or remove an older local tag; a successful build does not recreate the Workspace. Run `elyro up --recreate` when an existing Workspace should use the rebuilt tag.

## Execution

```bash
elyro shell
elyro exec -- go test ./...
elyro exec -- bash -lc 'go test ./... | tee /tmp/test.log'
```

`exec` passes argv and standard streams directly, does not allocate a TTY, and preserves the command exit code. Pipes and redirections require an explicit shell.

`shell` and `exec` require an existing Workspace. They never create, start, or replace one implicitly; call `elyro up` first.

Official Toolchain images use a small native zsh prompt that identifies the Workspace and current Linux directory, for example `elyro:demo ~/demo ❯`. The prompt, directory listings, autosuggestions, and syntax highlighting use terminal color only in an interactive shell. Set `NO_COLOR` before `elyro shell`, or use `TERM=dumb`, for a plain prompt. Custom images continue to own their shell and prompt configuration.

## Machine-readable inspection

```bash
elyro status --json
elyro list --json
elyro doctor --json
elyro doctor --project-dir PATH --json
```

Workspace lifecycle and inspection output uses schema 1 and exposes product concepts only: identity, project and mount paths, lifecycle status, Environment, Toolchain, image, platform, hostname, and published ports. Docker container names, labels, SSH aliases, identities, and known-hosts paths are implementation details and are not part of JSON output.

Doctor uses schema 2 with `kind`, `healthy`, an optional resolved `project`, and scoped checks. Each check has a stable `scope`, `name`, `status`, and non-empty `message`; any `fail` makes the command exit non-zero, while `warn` remains successful. Doctor automatically adds project checks when the current directory belongs to a configured, registered, Git, or detected Toolchain project. An unconfigured Git project with no detectable Toolchain is a warning; an invalid `elyro.yaml` is a failure. For a configured runtime environment, `project.runtime_environment` exposes sorted variable names and declared relative file paths, never values or a fingerprint. Doctor remains read-only. Errors use a non-zero exit code and actionable stderr; there is no global JSON error envelope.

## Terminal output

Human-facing commands use restrained semantic color and short completion receipts when stdout is a terminal. Brand, section, question, command, progress, and result styles have separate roles so color never carries the only meaning. Output is automatically plain when redirected, when `NO_COLOR` is non-empty, when `TERM=dumb`, or in CI. No color flag is required.

Machine contracts do not pass through the presentation layer: JSON stdout contains only JSON, `exec` and `shell` preserve command streams, and `skill show` prints the embedded source byte-for-byte.

## Skill

```bash
elyro skill show
elyro skill install codex|claude-code|all [--force]
elyro skill uninstall codex|claude-code|all [--force]
```

Install is idempotent for identical content. Different or modified content is protected unless `--force` is explicit.
