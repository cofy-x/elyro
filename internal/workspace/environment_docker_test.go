package workspace

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEnvironmentSupportsDockerOptions(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	socketPath := filepath.Join(projectDir, "tmp", "docker.sock")
	writeEnvironmentConfig(t, projectDir, `
environments:
  sandboxd:
    toolchain: go
    platform: linux/arm64
    docker:
      privileged: true
      publish:
        - "8080:8000"
      mounts:
        - source: ./tmp/docker.sock
          target: /var/run/docker.sock
        - source: ~/sandboxd-cache
          target: /var/cache/sandboxd
          read_only: true
`)

	environment, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{
		Environment:         "sandboxd",
		EnvironmentExplicit: true,
	})
	if err != nil {
		t.Fatalf("ResolveEnvironment returned error: %v", err)
	}
	if !environment.Docker.Privileged {
		t.Fatalf("expected environment to be privileged")
	}
	if got, want := NormalizePublishSpecs(environment.Docker.Publishes), "8080:8000"; got != want {
		t.Fatalf("environment publishes = %q, want %q", got, want)
	}
	if got, want := len(environment.Docker.Mounts), 2; got != want {
		t.Fatalf("mount count = %d, want %d", got, want)
	}
	var socketMount *DockerMount
	for i := range environment.Docker.Mounts {
		if environment.Docker.Mounts[i].Target == "/var/run/docker.sock" {
			socketMount = &environment.Docker.Mounts[i]
			break
		}
	}
	if socketMount == nil {
		t.Fatalf("docker socket mount missing from %#v", environment.Docker.Mounts)
	}
	if got, want := socketMount.Source, filepath.Clean(socketPath); got != want {
		t.Fatalf("mount source = %q, want %q", got, want)
	}
	if got := socketMount.ReadOnly; got {
		t.Fatalf("expected docker socket mount to be writable")
	}
	if got := NormalizeDockerMounts(environment.Docker.Mounts); !strings.Contains(got, "/var/run/docker.sock") || !strings.Contains(got, ":ro") {
		t.Fatalf("NormalizeDockerMounts = %q, expected both docker socket and read-only suffix", got)
	}
}

func TestResolveEnvironmentRejectsRelativeDockerTarget(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	writeEnvironmentConfig(t, projectDir, `
environments:
  sandboxd:
    image: example.com/team/sandboxd:latest
    docker:
      mounts:
        - source: ./tmp
          target: tmp
`)

	_, err := ResolveEnvironment(projectDir, "/home/elyro/demo", EnvironmentSelection{
		Environment:         "sandboxd",
		EnvironmentExplicit: true,
	})
	if err == nil || !strings.Contains(err.Error(), "docker.mounts[].target must be an absolute path") {
		t.Fatalf("expected absolute target validation error, got %v", err)
	}
}
