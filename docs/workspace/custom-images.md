# Custom Workspace Images

Elyro can manage a project-defined Workspace image without requiring that image to
inherit from an official Elyro image. This is the fully controlled path for an
individual developer who wants to own the operating system packages,
toolchains, update policy, and image build.

Official Elyro images remain the supported batteries-included path. A custom
image must satisfy the runtime contract below, while Elyro continues to manage
the project mount, container lifecycle, loopback SSH port, SSH key injection,
editor handoff, and workspace registration.

## Configure A Custom Image

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
image in this first workflow, including environments that also set `toolchain`; a
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

Docker `ENV` supplies the environment used by `elyro exec`. The image author is
also responsible for making Toolchain paths available to an SSH login shell,
for example through `/etc/environment` or the distribution's login profile.

The image may use any default Docker user. Elyro explicitly uses UID `0` only
for SSH configuration. `elyro shell` and `elyro exec` use `docker exec` as
`elyro`; editor and direct user SSH sessions also log in as `elyro`.

On Linux, bind-mounted project files must be writable by UID/GID `1000`. Elyro does not change host ownership or broaden project permissions automatically.

Elyro installs its managed public key in `/home/elyro/.ssh/authorized_keys` and
writes an SSH drop-in that enables public-key authentication while disabling
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
elyro down
elyro up
```

Start with the verified [from-scratch Ubuntu example](../../examples/workspace/custom-image-from-scratch/README.md). The existing [custom environment example](../../examples/workspace/go-custom-image-environment/README.md) demonstrates the alternative path of extending an official Elyro image.

The Ubuntu example uses official package services by default and accepts an
explicit `MIRROR_SOURCE` build argument for regional local builds. This option
changes only the example's package source; it does not weaken package signature
verification or change Elyro release defaults.
