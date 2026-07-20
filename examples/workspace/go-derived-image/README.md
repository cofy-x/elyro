# Go Project Workspace Image

This example is the public, Homebrew-user workflow for persisting an OS-level tool on top of Elyro's maintained Go Toolchain image. The project owns `.elyro/Dockerfile`; Elyro owns only input validation, the explicit Docker invocation, and Workspace lifecycle.

Install Elyro and build the project image:

```bash
brew install cofy-x/tap/elyro
cd examples/workspace/go-derived-image
elyro image build
```

Start the Workspace and verify both the project and the added `sqlite3` binary:

```bash
elyro up
elyro exec -- go run .
elyro exec -- sqlite3 :memory: 'select 42;'
```

After changing `.elyro/Dockerfile`, rebuild the same project tag and replace the existing Workspace explicitly:

```bash
elyro image build
elyro up --recreate
```

`elyro up` never builds the Dockerfile. A failed build leaves both the previous local image tag and any running Workspace available.
