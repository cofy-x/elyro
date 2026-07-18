# Node Workspace Image

This image adds the pinned Node.js LTS release, npm, npx, Corepack, Python, and the native-addon build toolchain to `workspace-base`.

Elyro does not select or globally install pnpm, Yarn, or Bun. Projects should declare their package manager with the `packageManager` field and let Corepack resolve it.

The native-addon contract is intentionally limited to `g++`, `make`, Python, the headers bundled with Node.js, and npm's bundled `node-gyp`; Elyro does not install the broader `build-essential` package set.
