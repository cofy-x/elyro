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
	record := workspace.Record{ContainerName: "container", ContainerWorkspaceDir: "/home/elyro/demo"}
	got := dockerShellArgs(record)
	for _, value := range []string{"-it", "--user", "elyro", "--workdir", "/home/elyro/demo", "container"} {
		if !slices.Contains(got, value) {
			t.Fatalf("docker shell args missing %q: %#v", value, got)
		}
	}
}
