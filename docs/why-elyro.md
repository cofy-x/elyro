# Why Elyro

Elyro solves one narrow problem: keep editing on macOS while running development commands in a predictable local Linux Workspace.

The default path is intentionally small:

```bash
elyro up --open
elyro exec -- go test ./...
```

Elyro is useful when a project needs Linux behavior, a repeatable Toolchain, or an editor connected to Linux without moving source ownership away from the Mac. Its stable JSON and argv-safe `exec` also give host coding agents a clear boundary for Linux validation.

## Deliberate trade-offs

- Elyro uses Docker and does not abstract multiple container engines.
- Official images favor a predictable Toolchain over arbitrary package composition.
- `elyro.yaml` covers project-specific ports, mounts, editor settings, platform, and custom images; zero-config projects do not need it.
- SSH exists for editor handoff and direct user access. Ordinary command execution uses local Docker directly.
- Elyro does not run coding agents, proxy models, manage credentials, orchestrate remote machines, or provide team infrastructure.

Elyro is not a replacement for the [Development Containers specification](https://containers.dev/), Docker Compose, a cloud development environment, or a security sandbox. Those tools solve broader portability, multi-service, remote-compute, or isolation problems. Elyro chooses a smaller Mac-to-local-Linux workflow so its behavior remains easy to understand and automate.

## Product test

A feature belongs in Elyro only when it makes the local Workspace lifecycle, Linux execution, editor handoff, diagnostics, or host-agent usage materially clearer. Features that create a second lifecycle or control plane stay outside the product.
