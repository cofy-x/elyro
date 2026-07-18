# Workspace Base

`workspace-base` is the direct, digest-pinned Ubuntu foundation for all official Elyro Toolchains. It provides user `elyro` with UID/GID 1000, `/home/elyro`, passwordless sudo, foreground SSH, zsh/bash, Git, curl, jq, procps, and the standard commands required by Workspace management.

It intentionally contains no Python, Go, Java, Node.js, or coding-agent CLI.

```bash
make workspace-base-image-build
make workspace-base-image-build-arm64
```

The image uses official Ubuntu sources and no proxy by default. Mirror and proxy build arguments exist only for explicit development and CI use.
