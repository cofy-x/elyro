# Workspace

A Workspace mounts one host project into a local Linux container. Elyro owns image selection, lifecycle, project mount, direct command execution, SSH editor access, and registry metadata.

```bash
elyro up --toolchain go
elyro status --json
elyro exec -- go test ./...
elyro shell
elyro open --editor cursor
elyro down
```

`shell` and `exec` use local Docker directly. SSH is configured only for Remote SSH and direct user access. The container hostname is the sanitized project name; the unique container name still contains a project-path hash.

`elyro init` creates `elyro.yaml` for named Environments and advanced settings. Without configuration, `up` detects a single Python, Go, Node.js, or Java Toolchain and writes no project file.

When `--project-dir` is omitted, Elyro uses the nearest `elyro.yaml`, then an existing containing Workspace, then the nearest Git root. This keeps every project-scoped command on the same Workspace when it runs from a subdirectory.

- [CLI reference](cli-reference.md)
- [Workspace configuration](configuration.md)
- [Custom images](custom-images.md)
- [Coding-agent workflow](../coding-agents.md)
