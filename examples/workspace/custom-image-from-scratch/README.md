# Custom Workspace Image From Scratch

This example builds a minimal Workspace image without using an Elyro image as its
base. Elyro manages the local container, project mount, SSH access, and editor
handoff; the Dockerfile remains under your control.

## Build And Run

From this directory:

```bash
docker build -t elyro/custom-workspace:local .
elyro up
elyro shell
```

The Dockerfile disables SSH password and root login, assigns the `elyro` account
a build-time random password that is never disclosed, and uses official Ubuntu
package services by default. Regional local builds can select the same explicit
mirrors supported by Elyro image builds:

```bash
docker build \
  --build-arg MIRROR_SOURCE=aliyun \
  --build-arg ELYRO_PROXY_URL=http://host.docker.internal:7890 \
  -t elyro/custom-workspace:local .
```

`MIRROR_SOURCE` accepts `official`, `aliyun`, `ustc`, or `tuna`.

Run a command without opening an interactive shell:

```bash
elyro exec -- git --version
```

The image contains only the packages required by the example and the Elyro
image contract. Add project toolchains and utilities to the Dockerfile, rebuild
the same image tag, and run `elyro down && elyro up` to recreate the
workspace from it.

The environment intentionally omits `toolchain`: a custom image does not inherit a
Elyro language toolchain or its editor defaults. Add explicit VS Code extensions
and settings to `elyro.yaml` when needed.

See the [custom Workspace image guide](../../../docs/workspace/custom-images.md) for
the complete distro-neutral runtime contract and ownership boundary.
