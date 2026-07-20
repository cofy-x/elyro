# Project-derived and Custom Workspace Images

The recommended way to persist project-specific OS libraries, compilers, database clients, or global CLIs is to derive a project image from an official Elyro Toolchain image:

```bash
elyro image init
# edit the commented package example in .elyro/Dockerfile
elyro image build
elyro up
```

The project owns the Dockerfile, package sources, versions, and update policy. Elyro validates the build inputs, invokes Docker without shell interpolation, and reports the result. It does not infer whether an image is stale, build during `up`, add packages itself, pass secrets or build arguments, publish the image, or recreate a running Workspace.

Use the [derived Go image example](../../examples/workspace/go-derived-image/README.md) for the complete Homebrew-installed workflow.

## Choose the right dependency mechanism

- Use `sudo apt` inside a Workspace for temporary investigation; it disappears when the Workspace is recreated or removed.
- Use the language's project dependency mechanism for linters, formatters, test runners, and application libraries.
- Use a derived Workspace image for persistent OS libraries, compilers, database clients, and global CLIs.
- Use Compose or another external tool for multi-service systems, and a Dev Container implementation when Features are required.

## Fully custom images

Elyro can also manage a project-defined Workspace image without requiring that image to
inherit from an official Elyro image. This is the fully controlled path for an
individual developer who wants to own the operating system packages,
toolchains, update policy, and image build.

Official Elyro images remain the supported batteries-included path. A custom
image must satisfy the runtime contract below, while Elyro continues to manage
the project mount, container lifecycle, loopback SSH port, SSH key injection,
editor handoff, and workspace registration.

## Configure a fully custom image

Build or pull the image before starting the Workspace, then reference it from a
project environment:

```yaml
version: 1
default_environment: custom
environments:
  custom:
    image: my-project-workspace:local
```

```bash
docker build -t my-project-workspace:local -f .elyro/Dockerfile .
elyro up
elyro shell
```

An image-only environment uses the host's default `linux/amd64` or `linux/arm64`
platform. Set `platform` explicitly only when the image and Docker host support
that target. Elyro does not build or automatically pull an explicitly configured
image without a `build` block, including environments that also set `toolchain`; a
missing image fails with an instruction to build or pull it first.

`toolchain` is optional when `image` is set. Adding a built-in toolchain selects that
toolchain's recommended editor settings, but it does not make the custom image
inherit the corresponding Elyro toolchain. Omit it for a fully independent
image and declare editor settings explicitly when needed.

## Image Contract

The contract is Linux distribution independent. A compatible image must:

- support the selected `linux/amd64` or `linux/arm64` platform
- contain an `elyro` user with UID/GID `1000` and home directory `/home/elyro`
- provide `bash`, OpenSSH `sshd`, procps-compatible `pgrep` and `pkill`, plus
  standard userland commands including `id`, `install`, `awk`, `mktemp`,
  `touch`, `chown`, `chmod`, `cat`, `rm`, `mv`, and `dirname` on `PATH`
- run an SSH daemon on container port `22` and keep the container process alive
- allow Docker to execute management commands as UID `0`
- allow UID `0` to create and own files below `/home/elyro` and `/etc/ssh`

Docker `ENV` supplies the lowest-priority environment used by `elyro exec`. A project may override it explicitly with [`docker.env_files` and `docker.environment`](configuration.md#runtime-environment). The image author is
also responsible for making Toolchain paths available to an SSH login shell,
for example through `/etc/environment` or the distribution's login profile.

Runtime environment values are container configuration, not build inputs: `elyro image build` never reads or forwards them. They are also not secrets because Docker users can inspect the resulting container configuration.

The image may use any default Docker user. Elyro explicitly uses UID `0` only
for SSH configuration. `elyro shell` and `elyro exec` use `docker exec` as
`elyro`; editor and direct user SSH sessions also log in as `elyro`.

On Linux, bind-mounted project files must be writable by UID/GID `1000`. Elyro does not change host ownership or broaden project permissions automatically.

Elyro installs its managed public key in `/home/elyro/.ssh/authorized_keys` and
writes an SSH drop-in that enables public-key authentication and the root-owned managed runtime environment file while disabling
password, keyboard-interactive, and root login. The image should start with
equivalent safe defaults rather than temporarily exposing password login.

The container SSH port is published on `127.0.0.1` only. Other application
ports remain private unless the project environment or `elyro up --publish`
exposes them.

## Ownership And Updates

For a custom image, you own:

- base-image selection, digest pinning, package sources, and security updates
- language toolchains, editor dependencies, and shell setup
- compatibility with every architecture declared by the environment
- rebuilding the image when its Dockerfile or dependencies change

Elyro does not publish, sign, scan, or update project-defined images. Rebuilding
the same tag does not replace an already running Workspace; recreate it explicitly:

```bash
docker build -t my-project-workspace:local -f .elyro/Dockerfile .
elyro up --dry-run --recreate
elyro up --recreate
```

When an Environment has a `build` block, use `elyro image build` instead. A fully custom image without that block continues to use its own build or pull command.

Start with the verified [from-scratch Ubuntu example](../../examples/workspace/custom-image-from-scratch/README.md). The existing [custom environment example](../../examples/workspace/go-custom-image-environment/README.md) demonstrates the alternative path of extending an official Elyro image.

The Ubuntu example uses official package services by default and accepts an
explicit `MIRROR_SOURCE` build argument for regional local builds. This option
changes only the example's package source; it does not weaken package signature
verification or change Elyro release defaults.
