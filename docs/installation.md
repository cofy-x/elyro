# Installation

Elyro supports macOS and Linux on amd64 and arm64. Docker is required. VS Code or Cursor with Remote SSH is optional.

Official Workspace images run development commands as UID/GID `1000`. On Linux, the project directory must be writable by that user; this matches the default first-user UID/GID on common desktop distributions. macOS users do not need to adjust project ownership because Docker Desktop mediates bind-mount access.

## Homebrew

```bash
brew install cofy-x/tap/elyro
elyro version --json
```

## Release installer

Download the repository installer and choose an exact release:

```bash
./scripts/install.sh --version v0.1.4
```

The archive contains one binary, `elyro`. The installer verifies `checksums.txt` before replacing it.

## Source

```bash
go install github.com/cofy-x/elyro/cmd/elyro@latest
```

## Uninstall

```bash
brew uninstall elyro
# or, for the release installer:
rm -f "${HOME}/.local/bin/elyro"
```

Installed Skills are user data and are removed explicitly:

```bash
elyro skill uninstall all
```
