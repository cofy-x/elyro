# Workspace

A Workspace mounts one host project into a local Linux container. Elyro owns image selection, lifecycle, project mount, direct command execution, SSH editor access, and registry metadata.

```bash
elyro up --dry-run --toolchain go
elyro up --toolchain go
elyro status --json
elyro exec -- go test ./...
elyro shell
elyro open --editor cursor
elyro down
```

`up --dry-run` explains whether the next lifecycle operation will create, start, reuse, or recreate the Workspace. `down --dry-run` identifies every Elyro-managed resource that would be removed and confirms that project files, mounted host data, and local images remain.

`shell` and `exec` use local Docker directly. SSH is configured only for Remote SSH and direct user access. The container hostname is the sanitized project name; the unique container name still contains a project-path hash.

The official images open a native zsh prompt such as `elyro:demo ~/demo ❯`, making the Workspace boundary and current Linux directory visible without adding a shell framework. `NO_COLOR=1 elyro shell` keeps the interactive prompt plain.

`elyro init` creates `elyro.yaml` for named Environments and advanced settings. Without configuration, `up` detects a single Python, Go, Node.js, or Java Toolchain and writes no project file.

Explicit `docker.environment` values and project-relative `docker.env_files` become container-wide runtime configuration inherited by `exec`, `shell`, SSH, and editor terminals. Elyro never reads `.env` or arbitrary host variables implicitly, and these Docker-visible values are not a secret store.

When `--project-dir` is omitted, Elyro uses the nearest `elyro.yaml`, then an existing containing Workspace, then the nearest Git root. This keeps every project-scoped command on the same Workspace when it runs from a subdirectory.

- [CLI reference](cli-reference.md)
- [Workspace configuration](configuration.md)
- [Custom images](custom-images.md)
- [Coding-agent workflow](../coding-agents.md)
