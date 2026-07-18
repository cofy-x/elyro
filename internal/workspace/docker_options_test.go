package workspace

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDockerMountHelpers(t *testing.T) {
	t.Parallel()

	mounts := []DockerMount{
		{Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
		{Source: "/tmp/cache", Target: "/var/cache/sandboxd", ReadOnly: true},
	}

	if got, want := NormalizeDockerMounts(mounts), "/var/run/docker.sock:/var/run/docker.sock,/tmp/cache:/var/cache/sandboxd:ro"; got != want {
		t.Fatalf("NormalizeDockerMounts = %q, want %q", got, want)
	}

	if got, want := DockerMountArgs(mounts), []string{
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-v", "/tmp/cache:/var/cache/sandboxd:ro",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("DockerMountArgs = %#v, want %#v", got, want)
	}
}

func TestUnsafeEnvironmentReasons(t *testing.T) {
	project := t.TempDir()
	reasons := UnsafeEnvironmentReasons(project, DockerOptions{
		Privileged: true,
		Mounts: []DockerMount{
			{Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
			{Source: filepath.Dir(project), Target: "/host"},
		},
	})
	if len(reasons) != 3 {
		t.Fatalf("reasons = %#v, want 3", reasons)
	}
}
