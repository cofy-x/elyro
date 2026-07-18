# Workspace Java

`elyro/workspace-java:dev-amd64` / `elyro/workspace-java:dev-arm64` are local Elyro
Workspace toolchain images for Java development.

## What It Provides

- Inherits `elyro/workspace-base:dev-<arch>` in source builds
- OpenJDK 21
- Maven
- Gradle
- All shared Workspace tooling from `workspace-base`

## Build

From the repository root:

```bash
make workspace-java-image-build
make workspace-java-image-build-arm64
```

The selected platform is tagged explicitly, for example
`elyro/workspace-java:dev-arm64`.
