package cli

import (
	"slices"
	"testing"

	"github.com/cofy-x/elyro/internal/workspace"
)

func TestDockerExecPreservesArgumentBoundaries(t *testing.T) {
	command := []string{"printf", "%s", "hello world", "what's up"}
	record := workspace.Record{ContainerName: "container", ContainerWorkspaceDir: "/home/elyro/demo"}
	got := dockerExecArgs(record, "/tmp/elyro-exec-test.pid", command)
	wantSuffix := []string{"elyro-exec", "/tmp/elyro-exec-test.pid", "printf", "%s", "hello world", "what's up"}
	if len(got) < len(wantSuffix) || !slices.Equal(got[len(got)-len(wantSuffix):], wantSuffix) {
		t.Fatalf("docker exec args suffix = %#v, want %#v", got, wantSuffix)
	}
	for _, value := range []string{"-i", "--user", "elyro", "--workdir", "/home/elyro/demo", "container"} {
		if !slices.Contains(got, value) {
			t.Fatalf("docker exec args missing %q: %#v", value, got)
		}
	}
}

func TestDockerShellUsesElyroProjectDirectoryAndTTY(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	record := workspace.Record{ContainerName: "container", ContainerWorkspaceDir: "/home/elyro/demo"}
	got := dockerShellArgs(record)
	for _, value := range []string{"-it", "--user", "elyro", "--workdir", "/home/elyro/demo", "container"} {
		if !slices.Contains(got, value) {
			t.Fatalf("docker shell args missing %q: %#v", value, got)
		}
	}
}

func TestDockerShellPassesOnlySupportedTerminalOverrides(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "dumb")
	t.Setenv("COLORTERM", "truecolor")

	got := dockerShellArgs(workspace.Record{ContainerName: "container", ContainerWorkspaceDir: "/home/elyro/demo"})
	for _, value := range []string{"NO_COLOR=1", "TERM=dumb"} {
		if !slices.Contains(got, value) {
			t.Fatalf("docker shell args missing %q: %#v", value, got)
		}
	}
	for _, value := range got {
		if value == "COLORTERM=truecolor" || value == "TERM=xterm-256color" {
			t.Fatalf("docker shell args pass unsupported host terminal value %q: %#v", value, got)
		}
	}
}
