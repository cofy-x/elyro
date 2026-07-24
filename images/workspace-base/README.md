# Workspace Base

`workspace-base` is the direct, digest-pinned Ubuntu foundation for all official Elyro Toolchains. It provides user `elyro` with UID/GID 1000, `/home/elyro`, passwordless sudo, foreground SSH, zsh/bash, Git, curl, jq, procps, and the standard commands required by Workspace management.

The interactive zsh environment uses a native `elyro:<project> <cwd> ❯` prompt, colored directory listings, autosuggestions, and syntax highlighting. It has no shell framework or prompt runtime dependency. `NO_COLOR` and `TERM=dumb` select the equivalent plain prompt. Build-time apt remains non-interactive, while the published image does not export `DEBIAN_FRONTEND`, so apt commands run by a developer retain their normal interactive behavior.

It intentionally contains no Python, Go, Node.js, or coding-agent CLI.

```bash
make workspace-base-image-build
make workspace-base-image-build-arm64
```

The image uses official Ubuntu sources and no proxy by default. Mirror and proxy build arguments exist only for explicit development and CI use.
