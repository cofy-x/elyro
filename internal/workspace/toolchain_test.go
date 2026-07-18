package workspace

import "testing"

func TestParseToolchain(t *testing.T) {
	t.Parallel()

	if _, err := ParseToolchain("python"); err != nil {
		t.Fatalf("ParseToolchain(python): %v", err)
	}
	if _, err := ParseToolchain("go"); err != nil {
		t.Fatalf("ParseToolchain(go): %v", err)
	}
	if _, err := ParseToolchain("java"); err != nil {
		t.Fatalf("ParseToolchain(java): %v", err)
	}
	if _, err := ParseToolchain("node"); err != nil {
		t.Fatalf("ParseToolchain(node): %v", err)
	}
	if _, err := ParseToolchain("rust"); err == nil {
		t.Fatal("ParseToolchain(rust) expected error")
	}
}

func TestToolchainDockerContextDir(t *testing.T) {
	t.Parallel()

	if got, want := ToolchainPython.DockerContextDir(), "images/workspace-python"; got != want {
		t.Fatalf("python context: got %q want %q", got, want)
	}
	if got, want := ToolchainGo.DockerContextDir(), "images/workspace-go"; got != want {
		t.Fatalf("go context: got %q want %q", got, want)
	}
	if got, want := ToolchainJava.DockerContextDir(), "images/workspace-java"; got != want {
		t.Fatalf("java context: got %q want %q", got, want)
	}
	if got, want := ToolchainNode.DockerContextDir(), "images/workspace-node"; got != want {
		t.Fatalf("node context: got %q want %q", got, want)
	}
}

func TestToolchainImage(t *testing.T) {
	t.Parallel()

	if got, want := ToolchainPython.Image("linux/amd64"), "ghcr.io/cofy-x/elyro/workspace-python:dev-amd64"; got != want {
		t.Fatalf("python image: got %q want %q", got, want)
	}
	if got, want := ToolchainGo.Image("linux/arm64"), "ghcr.io/cofy-x/elyro/workspace-go:dev-arm64"; got != want {
		t.Fatalf("go image: got %q want %q", got, want)
	}
	if got, want := ToolchainJava.Image("linux/amd64"), "ghcr.io/cofy-x/elyro/workspace-java:dev-amd64"; got != want {
		t.Fatalf("java image: got %q want %q", got, want)
	}
	if got, want := ToolchainNode.Image("linux/arm64"), "ghcr.io/cofy-x/elyro/workspace-node:dev-arm64"; got != want {
		t.Fatalf("node image: got %q want %q", got, want)
	}
}
